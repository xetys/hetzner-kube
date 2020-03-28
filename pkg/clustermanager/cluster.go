package clustermanager

import (
	"fmt"
	"strings"
	"time"

	"github.com/xetys/hetzner-kube/pkg"
)

const rewriteTpl = `cat /etc/kubernetes/%s | sed -e 's/server: https\(.*\)/server: https:\/\/127.0.0.1:16443/g' > /tmp/cp && mv /tmp/cp /etc/kubernetes/%s`

// KubernetesVersion indicate the kubernetes version managed by the current application
const KubernetesVersion = "1.18.0"

// Manager is the structure used to mange cluster
type Manager struct {
	nodes            []Node
	clusterName      string
	cloudInitFile    string
	eventService     EventService
	nodeCommunicator NodeCommunicator
	clusterProvider  ClusterProvider
	haEnabled        bool
	isolatedEtcd     bool
}

// KeepCerts is an enumeration for existing certificate handling during master install
type KeepCerts int

//
const (
	// NONE generate completely new certificates
	NONE KeepCerts = 0
	// CA generate certificates using existing authority
	CA KeepCerts = 1
	// ALL keep all certificates
	ALL KeepCerts = 2
)

// NewClusterManager create a new manager for the cluster
func NewClusterManager(provider ClusterProvider, nodeCommunicator NodeCommunicator, eventService EventService, name string, haEnabled bool, isolatedEtcd bool, cloudInitFile string) *Manager {
	manager := &Manager{
		clusterName:      name,
		haEnabled:        haEnabled,
		isolatedEtcd:     isolatedEtcd,
		cloudInitFile:    cloudInitFile,
		eventService:     eventService,
		nodeCommunicator: nodeCommunicator,
		clusterProvider:  provider,
		nodes:            provider.GetAllNodes(),
	}

	return manager
}

// NewClusterManagerFromCluster create a new manager from an existing cluster
func NewClusterManagerFromCluster(cluster Cluster, provider ClusterProvider, nodeCommunicator NodeCommunicator, eventService EventService) *Manager {
	return &Manager{
		clusterName:      cluster.Name,
		haEnabled:        cluster.HaEnabled,
		isolatedEtcd:     cluster.IsolatedEtcd,
		cloudInitFile:    cluster.CloudInitFile,
		eventService:     eventService,
		nodeCommunicator: nodeCommunicator,
		clusterProvider:  provider,
		nodes:            cluster.Nodes,
	}
}

// Cluster creates a Cluster object for further processing
func (manager *Manager) Cluster() Cluster {
	return Cluster{
		Name:              manager.clusterName,
		Nodes:             manager.nodes,
		HaEnabled:         manager.haEnabled,
		IsolatedEtcd:      manager.isolatedEtcd,
		CloudInitFile:     manager.cloudInitFile,
		NodeCIDR:          manager.clusterProvider.GetNodeCidr(),
		KubernetesVersion: KubernetesVersion,
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
			provisioner := NewNodeProvisioner(node, manager)
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

// SetupEncryptedNetwork setups an encrypted virtual network using wireguard
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
			err := manager.nodeCommunicator.WriteFile(node, "/etc/wireguard/wg0.conf", wireGuardConf, OwnerRead)
			if err != nil {
				errChan <- err
			}

			overlayRouteConf := GenerateOverlayRouteSystemdService(node)
			err = manager.nodeCommunicator.WriteFile(node, "/etc/systemd/system/overlay-route.service", overlayRouteConf, AllRead)
			if err != nil {
				errChan <- err
			}

			_, err = manager.nodeCommunicator.RunCmd(
				node,
				"systemctl enable wg-quick@wg0 && systemctl restart wg-quick@wg0"+
					" && systemctl enable overlay-route.service && systemctl restart overlay-route.service")
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

// InstallMasters installs the kubernetes control plane to master nodes
func (manager *Manager) InstallMasters(keepCerts KeepCerts) error {
	commands := []NodeCommand{
		{"kubeadm init", "kubectl version > /dev/null &> /dev/null || kubeadm init --ignore-preflight-errors=all --config /root/master-config.yaml"},
		{"configure kubectl", "rm -rf $HOME/.kube && mkdir -p $HOME/.kube && cp -i /etc/kubernetes/admin.conf $HOME/.kube/config && chown $(id -u):$(id -g) $HOME/.kube/config"},
		{"install canal", "kubectl apply -f https://docs.projectcalico.org/v3.10/manifests/canal.yaml"},
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

			var resetCommand string

			switch keepCerts {
			case NONE:
				resetCommand = "kubeadm reset -f && rm -rf /etc/kubernetes/pki && mkdir /etc/kubernetes/pki"
			case CA:
				resetCommand = "mkdir -p /root/pki && cp -r /etc/kubernetes/pki/* /root/pki && kubeadm reset -f && cp -r /root/pki/ca* /etc/kubernetes/pki"
			case ALL:
				resetCommand = "mkdir -p /root/pki && cp -r /etc/kubernetes/pki/* /root/pki && kubeadm reset -f && cp -r /root/pki/* /etc/kubernetes/pki"
			}

			_, err := manager.nodeCommunicator.RunCmd(node, resetCommand)
			if err != nil {
				return err
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
	masterConfig := GenerateMasterConfiguration(node, masterNodes, etcdNodes, manager.Cluster().KubernetesVersion)
	if err := manager.nodeCommunicator.WriteFile(node, "/root/master-config.yaml", masterConfig, AllRead); err != nil {
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

// InstallEtcdNodes installs the etcd cluster
func (manager *Manager) InstallEtcdNodes(nodes []Node, keepData bool) error {

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	for _, node := range nodes {
		numProcs++

		go func(node Node) {
			manager.etcdInstallStep(node, nodes, errChan, keepData)

			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProcs)
}

func (manager *Manager) etcdInstallStep(node Node, nodes []Node, errChan chan error, keepData bool) {
	commands := []NodeCommand{
		{"download etcd", "mkdir -p /opt/etcd && curl -L https://storage.googleapis.com/etcd/v3.3.11/etcd-v3.3.11-linux-amd64.tar.gz -o /opt/etcd-v3.3.11-linux-amd64.tar.gz"},
		{"install etcd", "tar xzvf /opt/etcd-v3.3.11-linux-amd64.tar.gz -C /opt/etcd --strip-components=1"},
		//{"configure etcd", "systemctl enable etcd.service && systemctl stop etcd.service && rm -rf /var/lib/etcd && systemctl start etcd.service"},
	}
	// set systemd service
	etcdSystemdService := GenerateEtcdSystemdService(node, nodes)
	err := manager.nodeCommunicator.WriteFile(node, "/etc/systemd/system/etcd.service", etcdSystemdService, AllRead)
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
	// configure etcd
	configureCommand := "systemctl enable etcd.service && systemctl stop etcd.service && rm -rf /var/lib/etcd && systemctl start etcd.service"
	if keepData {
		configureCommand = "systemctl enable etcd.service && systemctl stop etcd.service && systemctl start etcd.service"
	}
	manager.eventService.AddEvent(node.Name, "configure etcd")
	_, err = manager.nodeCommunicator.RunCmd(node, configureCommand)
	if err != nil {
		errChan <- err
	}
	if manager.isolatedEtcd {
		manager.eventService.AddEvent(node.Name, pkg.CompletedEvent)
	} else {
		manager.eventService.AddEvent(node.Name, "etcd configured")
	}
}

// InstallWorkers installs kubernetes workers to given nodes
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
				_, err := manager.nodeCommunicator.RunCmd(
					node,
					"for i in ip_vs ip_vs_rr ip_vs_wrr ip_vs_sh nf_conntrack_ipv4; do modprobe $i; done"+
						" && kubeadm reset -f && "+joinCommand)
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

// SetupHA installs the high-availability plane to cluster
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

	// set apiserver-count to number of masters
	apiServerCount := fmt.Sprintf("- --apiserver-count=%d\n    image: gcr.io/", len(masterNodes))
	for _, node := range masterNodes {
		manager.eventService.AddEvent(node.Name, "set api-server count")
		manager.nodeCommunicator.TransformFileOverNode(node, node, "/etc/kubernetes/manifests/kube-apiserver.yaml", func(in string) string {
			return strings.Replace(in, "image: gcr.io/", apiServerCount, 1)
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

// DeployLoadBalancer installs a client based load balancer for the master nodes to given nodes
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
