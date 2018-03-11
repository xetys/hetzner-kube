package clustermanager

import (
	"github.com/xetys/hetzner-kube/pkg"
	"time"
	"fmt"
	"strings"
)

func NewClusterManager(provider ClusterProvider, eventService EventService, name string, haEnabled bool, isolatedEtcd bool, selfHosted bool, cloudInitFile string) *Cluster {
	cluster := &Cluster{
		Nodes: provider.GetAllNodes(),
		eventService:eventService,
		Name:name,
		HaEnabled:haEnabled,
		IsolatedEtcd:isolatedEtcd,
		SelfHosted:selfHosted,
		CloudInitFile:cloudInitFile,
	}

	return cluster
}

func (cluster *Cluster) ProvisionNodes(nodes []Node) error {
	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	for _, node := range nodes {
		numProcs++
		go func(node Node) {
			cluster.eventService.AddEvent(node.Name, "install packages")
			_, err := cluster.nodeCommunicator.RunCmd(node, "wget -cO- https://raw.githubusercontent.com/xetys/hetzner-kube/master/install-docker-kubeadm.sh | bash -")

			if err != nil {
				errChan <- err
			}

			cluster.eventService.AddEvent(node.Name, "packages installed")

			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProcs)
}

func (cluster *Cluster) SetupEncryptedNetwork() error {
	nodes := cluster.Nodes
	// render a public/private key pair
	keyPairs := GenerateKeyPairs(nodes[0], len(nodes))

	for i, keyPair := range keyPairs {
		cluster.Nodes[i].WireGuardKeyPair = keyPair
	}

	nodes = cluster.Nodes

	// for each node, get specific IP and install it on node
	errChan := make(chan error)
	trueChan := make(chan bool)
	numProc := 0
	for _, node := range nodes {
		numProc++
		go func(node Node) {
			cluster.eventService.AddEvent(node.Name, "configure wireguard")
			wireGuardConf := GenerateWireguardConf(node, cluster.Nodes)
			err := cluster.nodeCommunicator.WriteFile(node, "/etc/wireguard/wg0.conf", wireGuardConf, false)
			if err != nil {
				errChan <- err
			}

			_, err = cluster.nodeCommunicator.RunCmd(node, "systemctl enable wg-quick@wg0 && systemctl restart wg-quick@wg0")

			if err != nil {
				errChan <- err
			}

			cluster.eventService.AddEvent(node.Name, "wireguard configured")
			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProc)
}


func (cluster *Cluster) InstallMasters() error {

	commands := []NodeCommand{
		{"kubeadm init", "kubeadm init --config /root/master-config.yaml"},
		{"configure kubectl", "rm -rf $HOME/.kube && mkdir -p $HOME/.kube && cp -i /etc/kubernetes/admin.conf $HOME/.kube/config && chown $(id -u):$(id -g) $HOME/.kube/config"},
		{"install flannel", "kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/v0.9.1/Documentation/kube-flannel.yml"},
		{"configure flannel", "kubectl -n kube-system patch ds kube-flannel-ds --type json -p '[{\"op\":\"add\",\"path\":\"/spec/template/spec/tolerations/-\",\"value\":{\"key\":\"node.cloudprovider.kubernetes.io/uninitialized\",\"value\":\"true\",\"effect\":\"NoSchedule\"}}]'"},
		//{"install hcloud integration", fmt.Sprintf("kubectl -n kube-system create secret generic hcloud --from-literal=token=%s", AppConf.CurrentContext.Token)},
		//{"deploy cloud controller manager", "kubectl apply -f  https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/v1.0.0.yaml"},
	}

	// inject custom commands
	commands = append(commands, cluster.clusterProvider.GetAdditionalMasterInstallCommands()...)

	var masterNode Node

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProc := 0
	numMaster := 0

	for _, node := range cluster.Nodes {

		if node.IsMaster {
			_, err := cluster.nodeCommunicator.RunCmd(node, "kubeadm reset")
			if err != nil {
				return nil
			}

			_, err = cluster.nodeCommunicator.RunCmd(node, "rm -rf /etc/kubernetes/pki && mkdir /etc/kubernetes/pki")
			if err != nil {
				return nil
			}
			if len(cluster.Nodes) == 1 {
				commands = append(commands, NodeCommand{"taint master", "kubectl taint nodes --all node-role.kubernetes.io/master-"})
			}

			if numMaster == 0 {
				masterNode = node
			}

			numProc++
			go func(node Node) {
				cluster.installMasterStep(node, numMaster, masterNode, commands, trueChan, errChan)
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

func (cluster *Cluster) installMasterStep(node Node, numMaster int, masterNode Node, commands []NodeCommand, trueChan chan bool, errChan chan error) {

	// create master-configuration
	var etcdNodes []Node
	if cluster.HaEnabled {
		if cluster.IsolatedEtcd {
			etcdNodes = cluster.clusterProvider.GetEtcdNodes()
		} else {
			etcdNodes = cluster.clusterProvider.GetMasterNodes()
		}
	}
	masterNodes := cluster.clusterProvider.GetMasterNodes()
	masterConfig := GenerateMasterConfiguration(node, masterNodes, etcdNodes)
	if err := cluster.nodeCommunicator.WriteFile(node, "/root/master-config.yaml", masterConfig, false); err != nil {
		errChan <- err
	}

	if numMaster > 0 {
		cluster.eventService.AddEvent(node.Name, "copy PKI")

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
			err := cluster.nodeCommunicator.CopyFileOverNode(masterNode, node, "/etc/kubernetes/pki/"+file)
			if err != nil {
				errChan <- err
			}
		}
	}

	for i, command := range commands {
		cluster.eventService.AddEvent(node.Name, command.eventName)
		_, err := cluster.nodeCommunicator.RunCmd(node, command.command)
		if err != nil {
			errChan <- err
		}

		if numMaster > 0 && i > 0 {
			break
		}
	}

	if !cluster.HaEnabled {
		cluster.eventService.AddEvent(node.Name, pkg.CompletedEvent)
	}

	trueChan <- true
}

func (cluster *Cluster) InstallEtcdNodes(nodes []Node) error {

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
			err := cluster.nodeCommunicator.WriteFile(node, "/etc/systemd/system/etcd.service", etcdSystemdService, false)
			if err != nil {
				errChan <- err
			}

			// install etcd
			for _, command := range commands {
				cluster.eventService.AddEvent(node.Name, command.eventName)
				_, err := cluster.nodeCommunicator.RunCmd(node, command.command)
				if err != nil {
					errChan <- err
				}
			}
			if cluster.IsolatedEtcd {
				cluster.eventService.AddEvent(node.Name, pkg.CompletedEvent)
			} else {
				cluster.eventService.AddEvent(node.Name, "etcd configured")
			}
			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProcs)
}

func (cluster *Cluster) InstallWorkers(nodes []Node) error {
	var joinCommand string
	// var masterNode Node
	// find master
	for _, node := range cluster.Nodes {
		if node.IsMaster {
			for tries := 0; ; tries++ {
				output, err := cluster.nodeCommunicator.RunCmd(node, "kubeadm token create --print-join-command")
				if tries < 10 && err != nil {
					return err
				} else {
					time.Sleep(2 * time.Second)
				}
				joinCommand = output
				break
			}
			// masterNode = node
			break
		}
	}

	// now let the nodes join
	for _, node := range nodes {
		if !node.IsMaster && !node.IsEtcd {
			cluster.eventService.AddEvent(node.Name, "registering node")
			if cluster.HaEnabled {
				// joinCommand = strings.Replace(joinCommand, "https://" + masterNode.IPAddress + ":6443", "https://127.0.0.1:16443", 1)
			}
			_, err := cluster.nodeCommunicator.RunCmd(node, "kubeadm reset && "+joinCommand)
			if err != nil {
				return err
			}

			if cluster.HaEnabled {
				time.Sleep(10 * time.Second) // we need some time until the kubelet.conf appears

				rewriteTpl := `cat /etc/kubernetes/%s | sed -e 's/server: https\(.*\)/server: https:\/\/127.0.0.1:16443/g' > /tmp/cp && mv /tmp/cp /etc/kubernetes/%s`
				kubeConfigs := []string{"kubelet.conf", "bootstrap-kubelet.conf"}

				cluster.eventService.AddEvent(node.Name, "rewrite kubeconfigs")
				for _, conf := range kubeConfigs {
					_, err := cluster.nodeCommunicator.RunCmd(node, fmt.Sprintf(rewriteTpl, conf, conf))
					if err != nil {
						return err
					}
				}
				_, err = cluster.nodeCommunicator.RunCmd(node, "systemctl restart docker && systemctl restart kubelet")
				if err != nil {
					return err
				}
			}

			cluster.eventService.AddEvent(node.Name, pkg.CompletedEvent)
		}
	}

	return nil
}

func (cluster *Cluster) SetupHA() error {
	// copy pki
	masterNode, err := cluster.clusterProvider.GetMasterNode()
	if err != nil {
		return err
	}

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	// deploy load balancer
	masterNodes := cluster.clusterProvider.GetMasterNodes()
	err = cluster.DeployLoadBalancer(cluster.Nodes)
	if err != nil {
		return err
	}

	// set apiserver-count to 3
	for _, node := range masterNodes {
		cluster.eventService.AddEvent(node.Name, "set api-server count")
		cluster.nodeCommunicator.TransformFileOverNode(node, node, "/etc/kubernetes/manifests/kube-apiserver.yaml", func(in string) string {
			return strings.Replace(in, "image: gcr.io/", "- --apiserver-count=3\n    image: gcr.io/", 1)
		})
	}

	cluster.eventService.AddEvent(masterNode.Name, "configuring kube-proxy")
	// update config-map for kube-proxy to lb
	proxyUpdateCmd := `kubectl get -n kube-system configmap/kube-proxy -o=yaml | sed -e 's/server: https\(.*\)/server: https:\/\/127.0.0.1:16443/g' | kubectl -n kube-system apply -f -`
	cluster.nodeCommunicator.RunCmd(*masterNode, proxyUpdateCmd)

	// delete proxy pods
	cluster.nodeCommunicator.RunCmd(*masterNode, "kubectl get pods --all-namespaces | grep proxy | awk '{print$2}' | xargs kubectl -n kube-system delete pod")

	// rewrite all kubeconfigs
	rewriteTpl := `cat /etc/kubernetes/%s | sed -e 's/server: https\(.*\)/server: https:\/\/127.0.0.1:16443/g' > /tmp/cp && mv /tmp/cp /etc/kubernetes/%s`
	kubeConfigs := []string{"kubelet.conf", "controller-manager.conf", "scheduler.conf"}

	numProcs = 0
	for _, node := range masterNodes {
		numProcs++

		go func(node Node) {
			cluster.eventService.AddEvent(node.Name, "rewrite kubeconfigs")
			for _, conf := range kubeConfigs {
				_, err := cluster.nodeCommunicator.RunCmd(node, fmt.Sprintf(rewriteTpl, conf, conf))
				if err != nil {
					errChan <- err
				}
			}
			_, err = cluster.nodeCommunicator.RunCmd(node, "systemctl restart docker && systemctl restart kubelet")
			if err != nil {
				errChan <- err
			}

			// wait for the apiserver to be back online
			cluster.eventService.AddEvent(node.Name, "wait for apiserver")
			_, err = cluster.nodeCommunicator.RunCmd(node, `until $(kubectl get node > /dev/null 2>/dev/null ); do echo "wait.."; sleep 1; done`)
			cluster.eventService.AddEvent(node.Name, pkg.CompletedEvent)

			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProcs)
}

func (cluster *Cluster) DeployLoadBalancer(nodes []Node) error {

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	masterNodes := cluster.clusterProvider.GetMasterNodes()
	masterIps := strings.Join(Nodes2IPs(masterNodes), " ")
	for _, node := range nodes {
		if !node.IsMaster && node.IsEtcd {
			continue
		}
		numProcs++
		go func(node Node) {
			cluster.eventService.AddEvent(node.Name, "deploy load balancer")
			// delete old if exists
			_, err := cluster.nodeCommunicator.RunCmd(node, `docker ps | grep master-lb | awk '{print "docker stop "$1" && docker rm "$1}' | sh`)
			if err != nil {
				errChan <- err
			}
			_, err = cluster.nodeCommunicator.RunCmd(node, fmt.Sprintf("docker run -d --name=master-lb --restart=always -p 16443:16443 xetys/k8s-master-lb %s", masterIps))
			if err != nil {
				errChan <- err
			}

			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProcs)
}
