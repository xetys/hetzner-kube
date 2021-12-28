package clustermanager

import (
	"fmt"
	"github.com/xetys/hetzner-kube/pkg"
	"log"
	"strings"
	"time"
)

const rewriteTpl = `cat /etc/kubernetes/%s | sed -e 's/server: https\(.*\)/server: https:\/\/127.0.0.1:16443/g' > /tmp/cp && mv /tmp/cp /etc/kubernetes/%s`

// Manager is the structure used to mange cluster
type Manager struct {
	nodes             []Node
	clusterName       string
	cloudInitFile     string
	eventService      EventService
	nodeCommunicator  NodeCommunicator
	clusterProvider   ClusterProvider
	WireguardEnabled  bool
	haEnabled         bool
	isolatedEtcd      bool
	kubernetesVersion string
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
func NewClusterManager(provider ClusterProvider, nodeCommunicator NodeCommunicator, eventService EventService, name string, haEnabled bool, isolatedEtcd bool, cloudInitFile string, kubernetesVersion string) *Manager {
	manager := &Manager{
		clusterName:       name,
		haEnabled:         haEnabled,
		WireguardEnabled:  false, // todo bring back wireguard
		isolatedEtcd:      isolatedEtcd,
		cloudInitFile:     cloudInitFile,
		eventService:      eventService,
		nodeCommunicator:  nodeCommunicator,
		clusterProvider:   provider,
		nodes:             provider.GetAllNodes(),
		kubernetesVersion: kubernetesVersion,
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
		KubernetesVersion: manager.kubernetesVersion,
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
		//{"sysctl settings", `printf '# Strict RPF mode as required by canal/Calico\nnet.ipv4.conf.default.rp_filter=1\nnet.ipv4.conf.all.rp_filter=1\n' >/etc/sysctl.d/50-canal-calico.conf && sysctl --load=/etc/sysctl.d/50-canal-calico.conf`},
		//{"kubeadm init", "kubectl version > /dev/null &> /dev/null || kubeadm init --ignore-preflight-errors=all --config /root/master-config.yaml"},
		//{"configure kubectl", "rm -rf $HOME/.kube && mkdir -p $HOME/.kube && cp -i /etc/kubernetes/admin.conf $HOME/.kube/config && chown $(id -u):$(id -g) $HOME/.kube/config"},
		//{"install canal", "kubectl apply -f https://docs.projectcalico.org/v3.16/manifests/canal.yaml"},
		{"start rke2 server", "systemctl start rke2-server.service"},
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

			var _ string

			resetCmd := "systemctl stop rke2-server.service && rke2-killall.sh && rm -rf /etc/rancher/ && rm -rf /var/lib/rancher"
			switch keepCerts {
			case CA:
				resetCmd = fmt.Sprintf(
					"mkdir -p /root/pki && cp -r /var/lib/rancher/rke2/server/tls/*-ca.* /root/pki && %s && mkdir -p /var/lib/rancher/rke2/server/tls && cp -r /root/pki/* /var/lib/rancher/rke2/server/tls",
					resetCmd,
				)
			case ALL:
				resetCmd = fmt.Sprintf(
					"mkdir -p /root/pki && cp -r /var/lib/rancher/rke2/server/tls/* /root/pki && %s && mkdir -p /var/lib/rancher/rke2/server/tls && cp -r /root/pki/* /var/lib/rancher/rke2/server/tls",
					resetCmd,
				)
			}

			_, err := manager.nodeCommunicator.RunCmd(node, resetCmd)
			if err != nil {
				return err
			}

			if numMaster == 0 {
				masterNode = node
			}

			numProc++
			go func(node Node) {
				manager.installMasterStep(node, numMaster, masterNode, commands, trueChan, errChan)
			}(node)

			// this was parallel once, but we need it now to happen sequentially
			select {
			case err := <-errChan:
				return err
			case <-trueChan:
				numProc--
				manager.eventService.AddEvent(node.Name, "waiting 60s for next master step")
				time.Sleep(60 * time.Second)
			}
			numMaster++
		}
	}

	return waitOrError(trueChan, errChan, &numProc)
}

// installs kubernetes control plane to a given node
func (manager *Manager) installMasterStep(node Node, numMaster int, masterNode Node, commands []NodeCommand, trueChan chan bool, errChan chan error) {
	if manager.haEnabled {
		var config string
		if numMaster == 0 {
			config = GenerateRke2FirstMasterConfiguration(manager.clusterProvider.GetMasterNodes())
		} else {
			nodeServerToken, err := manager.nodeCommunicator.RunCmd(masterNode, "cat /var/lib/rancher/rke2/server/node-token")
			if err != nil {
				errChan <- err
			}

			config = GenerateRke2SecondaryMasterConfiguration(nodeServerToken, manager.clusterProvider.GetMasterNodes())
		}

		_, err := manager.nodeCommunicator.RunCmd(
			node,
			"mkdir -p /etc/rancher/rke2")
		if err != nil {
			errChan <- err
		}

		err = manager.nodeCommunicator.WriteFile(node, "/etc/rancher/rke2/config.yaml", config, AllRead)
		if err != nil {
			errChan <- err
		}
	}

	for i, command := range commands {
		manager.eventService.AddEvent(node.Name, command.EventName)
		_, err := manager.nodeCommunicator.RunCmd(node, command.Command)
		if err != nil {
			if numMaster == 0 {
				errChan <- err
			} else {
				log.Println("an error occurred, but we will proceed")
				log.Println(err)
				errOut, _ := manager.nodeCommunicator.RunCmd(node, "systemctl status rke2-server.service")
				log.Println(errOut)
			}
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
	masterNode, err := manager.clusterProvider.GetMasterNode()
	if err != nil {
		return err
	}

	commands := []NodeCommand{
		//{"sysctl settings", `printf '# Strict RPF mode as required by canal/Calico\nnet.ipv4.conf.default.rp_filter=1\nnet.ipv4.conf.all.rp_filter=1\n' >/etc/sysctl.d/50-canal-calico.conf && sysctl --load=/etc/sysctl.d/50-canal-calico.conf`},
		{"start rke2 agent", "systemctl start rke2-agent.service"},
	}

	// create join command
	nodeServerToken, err := manager.nodeCommunicator.RunCmd(*masterNode, "cat /var/lib/rancher/rke2/server/node-token")
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
				server := "https://" + masterNode.IPAddress + ":9345"
				if manager.haEnabled {
					server = "https://" + masterNode.IPAddress + ":19345"
				}
				configContent := GenerateRke2AgentConfiguration(server, nodeServerToken)
				_, err := manager.nodeCommunicator.RunCmd(
					node,
					"mkdir -p /etc/rancher/rke2")
				if err != nil {
					errChan <- err
				}

				err = manager.nodeCommunicator.WriteFile(node, "/etc/rancher/rke2/config.yaml", configContent, AllRead)
				if err != nil {
					errChan <- err
				}

				// todo enable HA
				//if manager.haEnabled {
				//	time.Sleep(10 * time.Second) // we need some time until the kubelet.conf appears
				//
				//	kubeConfigs := []string{"kubelet.conf", "bootstrap-kubelet.conf"}
				//
				//	manager.eventService.AddEvent(node.Name, "rewrite kubeconfigs")
				//	for _, conf := range kubeConfigs {
				//		_, err := manager.nodeCommunicator.RunCmd(node, fmt.Sprintf(rewriteTpl, conf, conf))
				//		if err != nil {
				//			errChan <- err
				//		}
				//	}
				//	_, err = manager.nodeCommunicator.RunCmd(node, "systemctl restart docker && systemctl restart kubelet")
				//	if err != nil {
				//		errChan <- err
				//	}
				//}

				for _, command := range commands {
					manager.eventService.AddEvent(node.Name, command.EventName)
					_, err := manager.nodeCommunicator.RunCmd(node, command.Command)
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

func (manager *Manager) SetupHAProxyForAllNodes() error {
	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0

	masterNodes := manager.clusterProvider.GetMasterNodes()

	for _, node := range manager.nodes {
		numProcs++
		go func(node Node) {
			manager.eventService.AddEvent(node.Name, "deploy load balancer")
			haProxyCfg := GenerateHaProxyConfiguration(masterNodes)

			err := manager.nodeCommunicator.WriteFile(node, "/etc/haproxy/haproxy.cfg", haProxyCfg, AllRead)
			if err != nil {
				errChan <- err
			}

			_, err = manager.nodeCommunicator.RunCmd(node, "systemctl restart haproxy")

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
