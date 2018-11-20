package clustermanager

import (
	"fmt"
	"strings"
	"time"

	"github.com/xetys/hetzner-kube/pkg"
)

const rewriteTpl = `cat /etc/kubernetes/%s | sed -e 's/server: https\(.*\)/server: https:\/\/127.0.0.1:16443/g' > /tmp/cp && mv /tmp/cp /etc/kubernetes/%s`

//Manager is the structure used to mange cluster
type Manager struct {
	nodes            []Node
	clusterName      string
	cloudInitFile    string
	k8sVersion       string
	eventService     EventService
	nodeCommunicator NodeCommunicator
	clusterProvider  ClusterProvider
	haEnabled        bool
	isolatedEtcd     bool
	selfHosted       bool
}

//NewClusterManager create a new manager for the cluster
func NewClusterManager(provider ClusterProvider, nodeCommunicator NodeCommunicator, eventService EventService, name string, haEnabled bool, isolatedEtcd bool, cloudInitFile string, selfHosted bool, k8sVersion string) *Manager {
	manager := &Manager{
		clusterName:      name,
		haEnabled:        haEnabled,
		isolatedEtcd:     isolatedEtcd,
		cloudInitFile:    cloudInitFile,
		selfHosted:       selfHosted,
		eventService:     eventService,
		nodeCommunicator: nodeCommunicator,
		clusterProvider:  provider,
		nodes:            provider.GetAllNodes(),
		k8sVersion:       k8sVersion,
	}

	return manager
}

//NewClusterManagerFromCluster create a new manager from an existing cluster
func NewClusterManagerFromCluster(cluster Cluster, provider ClusterProvider, nodeCommunicator NodeCommunicator, eventService EventService) *Manager {
	return &Manager{
		clusterName:      cluster.Name,
		haEnabled:        cluster.HaEnabled,
		isolatedEtcd:     cluster.IsolatedEtcd,
		cloudInitFile:    cluster.CloudInitFile,
		selfHosted:       cluster.SelfHosted,
		eventService:     eventService,
		nodeCommunicator: nodeCommunicator,
		clusterProvider:  provider,
		nodes:            cluster.Nodes,
	}
}

//Cluster creates a Cluster object for further processing
func (manager *Manager) Cluster() Cluster {
	return Cluster{
		Name:          manager.clusterName,
		Nodes:         manager.nodes,
		HaEnabled:     manager.haEnabled,
		IsolatedEtcd:  manager.isolatedEtcd,
		CloudInitFile: manager.cloudInitFile,
		SelfHosted:    manager.selfHosted,
		NodeCIDR:      manager.clusterProvider.GetNodeCidr(),
	}
}

// AppendNodes can be used to append nodes to the cluster after initialization
func (manager *Manager) AppendNodes(nodes []Node) {
	manager.nodes = append(manager.nodes, nodes...)
}

// ProvisionNodes install packages for the nodes
func (manager *Manager) ProvisionNodes(nodes []Node) error {
	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	for _, node := range nodes {
		numProcs++
		go func(node Node) {
			manager.eventService.AddEvent(node.Name, "install packages")
			//_, err := manager.nodeCommunicator.RunCmd(node, "wget -cO- https://raw.githubusercontent.com/xetys/hetzner-kube/master/install-docker-kubeadm.sh | bash -")
			provisioner := NewNodeProvisioner(node, manager.nodeCommunicator, manager.eventService, manager.k8sVersion)
			err := provisioner.Provision(node, manager.nodeCommunicator, manager.eventService)
			if err != nil {
				errChan <- err
			}

			manager.eventService.AddEvent(node.Name, "packages installed")

			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProcs)
}

//SetupEncryptedNetwork setups an encrypted virtual network using wireguard
// modifies the state of manager.Nodes
func (manager *Manager) SetupEncryptedNetwork() error {
	var err error
	var keyPair WgKeyPair

	for i := range manager.nodes {
		keyPair, err = GenerateKeyPair()
		if err != nil {
			return fmt.Errorf("unable to setup encrypted network: %v", err)
		}

		manager.nodes[i].WireGuardKeyPair = keyPair
	}

	nodes := manager.nodes

	// for each node, get specific IP and install it on node
	errChan := make(chan error)
	trueChan := make(chan bool)
	numProc := 0
	for _, node := range nodes {
		numProc++
		go func(node Node) {
			manager.eventService.AddEvent(node.Name, "configure wireguard")
			wireGuardConf := GenerateWireguardConf(node, manager.nodes)
			err := manager.nodeCommunicator.WriteFile(node, "/etc/wireguard/wg0.conf", wireGuardConf, false)
			if err != nil {
				errChan <- err
			}

			_, err = manager.nodeCommunicator.RunCmd(node, "systemctl enable wg-quick@wg0 && systemctl restart wg-quick@wg0")

			if err != nil {
				errChan <- err
			}

			manager.eventService.AddEvent(node.Name, "wireguard configured")
			trueChan <- true
		}(node)
	}

	err = waitOrError(trueChan, errChan, &numProc)
	if err != nil {
		return err
	}
	manager.clusterProvider.SetNodes(manager.nodes)
	return nil
}

//InstallMasters installs the kubernetes control plane to master nodes
func (manager *Manager) InstallMasters() error {

	commands := []NodeCommand{
		{"kubeadm init", "kubeadm init --config /root/master-config.yaml"},
		{"configure kubectl", "rm -rf $HOME/.kube && mkdir -p $HOME/.kube && cp -i /etc/kubernetes/admin.conf $HOME/.kube/config && chown $(id -u):$(id -g) $HOME/.kube/config"},
		{"install flannel", "kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/v0.10.0/Documentation/kube-flannel.yml"},
		{"configure flannel", "kubectl -n kube-system patch ds kube-flannel-ds --type json -p '[{\"op\":\"add\",\"path\":\"/spec/template/spec/tolerations/-\",\"value\":{\"key\":\"node.cloudprovider.kubernetes.io/uninitialized\",\"value\":\"true\",\"effect\":\"NoSchedule\"}}]'"},
		//{"install hcloud integration", fmt.Sprintf("kubectl -n kube-system create secret generic hcloud --from-literal=token=%s", AppConf.CurrentContext.Token)},
		//{"deploy cloud controller manager", "kubectl apply -f  https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/v1.0.0.yaml"},
	}

	// inject custom commands
	commands = append(commands, manager.clusterProvider.GetAdditionalMasterInstallCommands()...)

	var masterNode Node

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProc := 0
	numMaster := 0

	for _, node := range manager.nodes {
		if node.IsMaster {
			_, err := manager.nodeCommunicator.RunCmd(node, "kubeadm reset")
			if err != nil {
				return nil
			}

			_, err = manager.nodeCommunicator.RunCmd(node, "rm -rf /etc/kubernetes/pki && mkdir /etc/kubernetes/pki")
			if err != nil {
				return nil
			}
			if len(manager.nodes) == 1 {
				commands = append(commands, NodeCommand{"taint master", "kubectl taint nodes --all node-role.kubernetes.io/master-"})
			}

			if numMaster == 0 {
				masterNode = node
			}

			numProc++
			go func(node Node) {
				manager.installMasterStep(node, numMaster, masterNode, commands, trueChan, errChan)
			}(node)

			// early wait the first time
			if numMaster == 0 {
				select {
				case err := <-errChan:
					return err
				case <-trueChan:
					numProc--
				}
			}
			numMaster++
		}
	}

	return waitOrError(trueChan, errChan, &numProc)
}

// installs kubernetes control plane to a given node
func (manager *Manager) installMasterStep(node Node, numMaster int, masterNode Node, commands []NodeCommand, trueChan chan bool, errChan chan error) {

	// create master-configuration
	var etcdNodes []Node
	if manager.haEnabled {
		if manager.isolatedEtcd {
			etcdNodes = manager.clusterProvider.GetEtcdNodes()
		} else {
			etcdNodes = manager.clusterProvider.GetMasterNodes()
		}
	}
	masterNodes := manager.clusterProvider.GetMasterNodes()
	masterConfig := GenerateMasterConfiguration(node, masterNodes, etcdNodes)
	if err := manager.nodeCommunicator.WriteFile(node, "/root/master-config.yaml", masterConfig, false); err != nil {
		errChan <- err
	}

	if numMaster > 0 {
		manager.eventService.AddEvent(node.Name, "copy PKI")

		files := []string{
			"apiserver-kubelet-client.crt",
			"apiserver-kubelet-client.key",
			"apiserver.crt",
			"apiserver.key",
			"ca.crt",
			"ca.key",
			"front-proxy-ca.crt",
			"front-proxy-ca.key",
			"front-proxy-client.crt",
			"front-proxy-client.key",
			"sa.key",
			"sa.pub",
		}

		for _, file := range files {
			err := manager.nodeCommunicator.CopyFileOverNode(masterNode, node, "/etc/kubernetes/pki/"+file)
			if err != nil {
				errChan <- err
			}
		}
	}

	for i, command := range commands {
		manager.eventService.AddEvent(node.Name, command.EventName)
		_, err := manager.nodeCommunicator.RunCmd(node, command.Command)
		if err != nil {
			errChan <- err
		}

		if numMaster > 0 && i > 0 {
			break
		}
	}

	if !manager.haEnabled {
		manager.eventService.AddEvent(node.Name, pkg.CompletedEvent)
	}

	trueChan <- true
}

//InstallEtcdNodes installs the etcd cluster
func (manager *Manager) InstallEtcdNodes(nodes []Node) error {

	commands := []NodeCommand{
		{"download etcd", "mkdir -p /opt/etcd && curl -L https://storage.googleapis.com/etcd/v3.2.13/etcd-v3.2.13-linux-amd64.tar.gz -o /opt/etcd-v3.2.13-linux-amd64.tar.gz"},
		{"install etcd", "tar xzvf /opt/etcd-v3.2.13-linux-amd64.tar.gz -C /opt/etcd --strip-components=1"},
		{"configure etcd", "systemctl enable etcd.service && systemctl stop etcd.service && rm -rf /var/lib/etcd && systemctl start etcd.service"},
	}

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	for _, node := range nodes {
		numProcs++

		go func(node Node) {
			// set systemd service
			etcdSystemdService := GenerateEtcdSystemdService(node, nodes)
			err := manager.nodeCommunicator.WriteFile(node, "/etc/systemd/system/etcd.service", etcdSystemdService, false)
			if err != nil {
				errChan <- err
			}

			// install etcd
			for _, command := range commands {
				manager.eventService.AddEvent(node.Name, command.EventName)
				_, err := manager.nodeCommunicator.RunCmd(node, command.Command)
				if err != nil {
					errChan <- err
				}
			}
			if manager.isolatedEtcd {
				manager.eventService.AddEvent(node.Name, pkg.CompletedEvent)
			} else {
				manager.eventService.AddEvent(node.Name, "etcd configured")
			}
			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProcs)
}

//InstallWorkers installs kubernetes workers to given nodes
func (manager *Manager) InstallWorkers(nodes []Node) error {
	node, err := manager.clusterProvider.GetMasterNode()
	if err != nil {
		return err
	}

	// create join command
	joinCommand, err := manager.nodeCommunicator.RunCmd(*node, "kubeadm token create --print-join-command")
	if err != nil {
		return err
	}

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	// now let the nodes join
	for _, node := range nodes {
		if !node.IsMaster && !node.IsEtcd {
			numProcs++
			go func(node Node) {
				manager.eventService.AddEvent(node.Name, "registering node")
				_, err := manager.nodeCommunicator.RunCmd(node, "kubeadm reset && "+joinCommand)
				if err != nil {
					errChan <- err
				}
				if manager.haEnabled {
					time.Sleep(10 * time.Second) // we need some time until the kubelet.conf appears

					kubeConfigs := []string{"kubelet.conf", "bootstrap-kubelet.conf"}

					manager.eventService.AddEvent(node.Name, "rewrite kubeconfigs")
					for _, conf := range kubeConfigs {
						_, err := manager.nodeCommunicator.RunCmd(node, fmt.Sprintf(rewriteTpl, conf, conf))
						if err != nil {
							errChan <- err
						}
					}
					_, err = manager.nodeCommunicator.RunCmd(node, "systemctl restart docker && systemctl restart kubelet")
					if err != nil {
						errChan <- err
					}
				}
				manager.eventService.AddEvent(node.Name, pkg.CompletedEvent)
				trueChan <- true
			}(node)
		}
	}
	return waitOrError(trueChan, errChan, &numProcs)
}

//SetupHA installs the high-availability plane to cluster
func (manager *Manager) SetupHA() error {
	// copy pki
	masterNode, err := manager.clusterProvider.GetMasterNode()
	if err != nil {
		return err
	}

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	// deploy load balancer
	masterNodes := manager.clusterProvider.GetMasterNodes()
	err = manager.DeployLoadBalancer(manager.nodes)
	if err != nil {
		return err
	}

	// set apiserver-count to 3
	for _, node := range masterNodes {
		manager.eventService.AddEvent(node.Name, "set api-server count")
		manager.nodeCommunicator.TransformFileOverNode(node, node, "/etc/kubernetes/manifests/kube-apiserver.yaml", func(in string) string {
			return strings.Replace(in, "image: gcr.io/", "- --apiserver-count=3\n    image: gcr.io/", 1)
		})
	}

	manager.eventService.AddEvent(masterNode.Name, "configuring kube-proxy")
	// update config-map for kube-proxy to lb
	proxyUpdateCmd := `kubectl get -n kube-system configmap/kube-proxy -o=yaml | sed -e 's/server: https\(.*\)/server: https:\/\/127.0.0.1:16443/g' | kubectl -n kube-system apply -f -`
	manager.nodeCommunicator.RunCmd(*masterNode, proxyUpdateCmd)

	// delete proxy pods
	manager.nodeCommunicator.RunCmd(*masterNode, "kubectl get pods --all-namespaces | grep proxy | awk '{print$2}' | xargs kubectl -n kube-system delete pod")

	// rewrite all kubeconfigs
	kubeConfigs := []string{"kubelet.conf", "controller-manager.conf", "scheduler.conf"}

	numProcs = 0
	for _, node := range masterNodes {
		numProcs++

		go func(node Node) {
			manager.eventService.AddEvent(node.Name, "rewrite kubeconfigs")
			for _, conf := range kubeConfigs {
				_, err := manager.nodeCommunicator.RunCmd(node, fmt.Sprintf(rewriteTpl, conf, conf))
				if err != nil {
					errChan <- err
				}
			}
			_, err = manager.nodeCommunicator.RunCmd(node, "systemctl restart docker && systemctl restart kubelet")
			if err != nil {
				errChan <- err
			}

			// wait for the apiserver to be back online
			manager.eventService.AddEvent(node.Name, "wait for apiserver")
			_, err = manager.nodeCommunicator.RunCmd(node, `until $(kubectl get node > /dev/null 2>/dev/null ); do echo "wait.."; sleep 1; done`)
			manager.eventService.AddEvent(node.Name, pkg.CompletedEvent)

			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProcs)
}

//DeployLoadBalancer installs a client based load balancer for the master nodes to given nodes
func (manager *Manager) DeployLoadBalancer(nodes []Node) error {

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	masterNodesIP := []string{}
	for _, node := range manager.clusterProvider.GetMasterNodes() {
		masterNodesIP = append(masterNodesIP, node.IPAddress)
	}

	masterIps := strings.Join(masterNodesIP, " ")
	for _, node := range nodes {
		if !node.IsMaster && node.IsEtcd {
			continue
		}
		numProcs++
		go func(node Node) {
			manager.eventService.AddEvent(node.Name, "deploy load balancer")
			// delete old if exists
			_, err := manager.nodeCommunicator.RunCmd(node, `docker ps | grep master-lb | awk '{print "docker stop "$1" && docker rm "$1}' | sh`)
			if err != nil {
				errChan <- err
			}
			_, err = manager.nodeCommunicator.RunCmd(node, fmt.Sprintf("docker run -d --name=master-lb --restart=always -p 16443:16443 xetys/k8s-master-lb %s", masterIps))
			if err != nil {
				errChan <- err
			}

			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProcs)
}
