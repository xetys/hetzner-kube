package cmd

import (
	"errors"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/xetys/hetzner-kube/pkg"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

func (cluster *Cluster) CreateNodes(suffix string, template Node, datacenters []string, count int, offset int) ([]Node, error) {
	sshKey, _, err := AppConf.Client.SSHKey.Get(AppConf.Context, template.SSHKeyName)

	if err != nil {
		return nil, err
	}

	serverNameTemplate := fmt.Sprintf("%s-%s-@idx", cluster.Name, suffix)
	serverOptsTemplate := hcloud.ServerCreateOpts{
		Name: serverNameTemplate,
		ServerType: &hcloud.ServerType{
			Name: template.Type,
		},
		Image: &hcloud.Image{
			Name: "ubuntu-16.04",
		},
	}

	if len(cluster.CloudInitFile) > 0 {
		buf, err := ioutil.ReadFile(cluster.CloudInitFile)
		if err == nil {
			serverOptsTemplate.UserData = string(buf)
		}

	}

	serverOptsTemplate.SSHKeys = append(serverOptsTemplate.SSHKeys, sshKey)

	datacentersCount := len(datacenters)

	var nodes []Node
	for i := 1; i <= count; i++ {
		var serverOpts hcloud.ServerCreateOpts
		serverOpts = serverOptsTemplate
		nodeNumber := i + offset
		serverOpts.Name = strings.Replace(serverNameTemplate, "@idx", fmt.Sprintf("%.02d", nodeNumber), 1)
		serverOpts.Datacenter = &hcloud.Datacenter{
			Name: datacenters[i%datacentersCount],
		}

		// try to find

		// create
		server, err := cluster.runCreateServer(&serverOpts)

		if err != nil {
			return nil, err
		}

		ipAddress := server.Server.PublicNet.IPv4.IP.String()
		log.Printf("Created node '%s' with IP %s", server.Server.Name, ipAddress)
		privateIpLastBlock := nodeNumber
		if !template.IsEtcd {
			privateIpLastBlock += 10
			if !template.IsMaster {
				privateIpLastBlock += 10
			}
		}
		privateIpAddress := fmt.Sprintf("10.0.1.%d", privateIpLastBlock)

		node := Node{
			Name:             serverOpts.Name,
			Type:             serverOpts.ServerType.Name,
			IsMaster:         template.IsMaster,
			IsEtcd:           template.IsEtcd,
			IPAddress:        ipAddress,
			PrivateIPAddress: privateIpAddress,
			SSHKeyName:       template.SSHKeyName,
		}
		nodes = append(nodes, node)
		cluster.Nodes = append(cluster.Nodes, node)
	}

	return nodes, nil
}

func (cluster *Cluster) ProvisionNodes(nodes []Node) error {
	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	for _, node := range nodes {
		numProcs++
		go func(node Node) {
			cluster.coordinator.AddEvent(node.Name, "install packages")
			_, err := runCmd(node, "wget -cO- https://raw.githubusercontent.com/xetys/hetzner-kube/master/install-docker-kubeadm.sh | bash -")

			if err != nil {
				errChan <- err
			}

			cluster.coordinator.AddEvent(node.Name, "packages installed")

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
			cluster.coordinator.AddEvent(node.Name, "configure wireguard")
			wireGuardConf := GenerateWireguardConf(node, cluster.Nodes)
			err := writeNodeFile(node, "/etc/wireguard/wg0.conf", wireGuardConf, false)
			if err != nil {
				errChan <- err
			}

			_, err = runCmd(node, "systemctl enable wg-quick@wg0 && systemctl restart wg-quick@wg0")

			if err != nil {
				errChan <- err
			}

			cluster.coordinator.AddEvent(node.Name, "wireguard configured")
			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProc)
}

func (cluster *Cluster) runCreateServer(opts *hcloud.ServerCreateOpts) (*hcloud.ServerCreateResult, error) {

	log.Printf("creating server '%s'...", opts.Name)
	server, _, err := AppConf.Client.Server.GetByName(AppConf.Context, opts.Name)
	if err != nil {
		return nil, err
	}
	if server == nil {
		result, _, err := AppConf.Client.Server.Create(AppConf.Context, *opts)
		if err != nil {
			if err.(hcloud.Error).Code == "uniqueness_error" {
				server, _, err := AppConf.Client.Server.Get(AppConf.Context, opts.Name)

				if err != nil {
					return nil, err
				}

				return &hcloud.ServerCreateResult{Server: server}, nil
			}

			return nil, err
		}

		if err := AppConf.ActionProgress(AppConf.Context, result.Action); err != nil {
			return nil, err
		}

		cluster.wait = true

		return &result, nil
	} else {
		log.Printf("loading server '%s'...", opts.Name)
		return &hcloud.ServerCreateResult{Server: server}, nil
	}
}

func (cluster *Cluster) GetMasterNode() (node *Node, err error) {

	for _, node := range cluster.Nodes {
		if node.IsMaster {
			return &node, nil
		}
	}

	return nil, errors.New("no master node found")
}

func (cluster *Cluster) GetEtcdNodes() []Node {
	nodes := []Node{}
	for _, node := range cluster.Nodes {
		if node.IsEtcd {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func (cluster *Cluster) GetMasterNodes() []Node {
	nodes := []Node{}
	for _, node := range cluster.Nodes {
		if node.IsMaster {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func (cluster *Cluster) GetWorkerNodes() []Node {
	nodes := []Node{}
	for _, node := range cluster.Nodes {
		if !node.IsMaster && !node.IsEtcd {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func Node2IP(node Node) string {
	return node.IPAddress
}

func Nodes2IPs(nodes []Node) []string {
	ips := []string{}
	for _, node := range nodes {
		ips = append(ips, Node2IP(node))
	}

	return ips
}

func (cluster *Cluster) CreateEtcdNodes(sshKeyName string, masterServerType string, datacenters []string, count int) error {
	template := Node{SSHKeyName: sshKeyName, IsEtcd: true, Type: masterServerType}
	_, err := cluster.CreateNodes("etcd", template, datacenters, count, 0)
	saveCluster(cluster)
	return err
}

func (cluster *Cluster) CreateMasterNodes(sshKeyName string, masterServerType string, datacenters []string, count int) error {
	isEtcd := true
	if cluster.IsolatedEtcd {
		isEtcd = false
	}
	template := Node{SSHKeyName: sshKeyName, IsMaster: true, Type: masterServerType, IsEtcd: isEtcd}
	_, err := cluster.CreateNodes("master", template, datacenters, count, 0)
	saveCluster(cluster)
	return err
}

func (cluster *Cluster) CreateWorkerNodes(sshKeyName string, workerServerType string, datacenters []string, count int, offset int) ([]Node, error) {
	template := Node{SSHKeyName: sshKeyName, IsMaster: false, Type: workerServerType}
	nodes, err := cluster.CreateNodes("worker", template, datacenters, count, offset)
	saveCluster(cluster)
	return nodes, err
}

func (cluster *Cluster) InstallMasters() error {

	commands := []SSHCommand{
		{"kubeadm init", "kubeadm init --config /root/master-config.yaml"},
		{"configure kubectl", "rm -rf $HOME/.kube && mkdir -p $HOME/.kube && cp -i /etc/kubernetes/admin.conf $HOME/.kube/config && chown $(id -u):$(id -g) $HOME/.kube/config"},
		{"install flannel", "kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml"},
		{"configure flannel", "kubectl -n kube-system patch ds kube-flannel-ds --type json -p '[{\"op\":\"add\",\"path\":\"/spec/template/spec/tolerations/-\",\"value\":{\"key\":\"node.cloudprovider.kubernetes.io/uninitialized\",\"value\":\"true\",\"effect\":\"NoSchedule\"}}]'"},
		{"install hcloud integration", fmt.Sprintf("kubectl -n kube-system create secret generic hcloud --from-literal=token=%s", AppConf.CurrentContext.Token)},
		{"deploy cloud controller manager", "kubectl apply -f  https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/v1.0.0.yaml"},
	}
	var masterNode Node

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProc := 0
	numMaster := 0

	for _, node := range cluster.Nodes {

		if node.IsMaster {
			_, err := runCmd(node, "kubeadm reset")
			if err != nil {
				return nil
			}

			_, err = runCmd(node, "rm -rf /etc/kubernetes/pki && mkdir /etc/kubernetes/pki")
			if err != nil {
				return nil
			}
			if len(cluster.Nodes) == 1 {
				commands = append(commands, SSHCommand{"taint master", "kubectl taint nodes --all node-role.kubernetes.io/master-"})
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

func (cluster *Cluster) installMasterStep(node Node, numMaster int, masterNode Node, commands []SSHCommand, trueChan chan bool, errChan chan error) {

	// create master-configuration
	var etcdNodes []Node
	if cluster.HaEnabled {
		if cluster.IsolatedEtcd {
			etcdNodes = cluster.GetEtcdNodes()
		} else {
			etcdNodes = cluster.GetMasterNodes()
		}
	}
	masterNodes := cluster.GetMasterNodes()
	masterConfig := GenerateMasterConfiguration(node, masterNodes, etcdNodes)
	if err := writeNodeFile(node, "/root/master-config.yaml", masterConfig, false); err != nil {
		errChan <- err
	}

	if numMaster > 0 {
		cluster.coordinator.AddEvent(node.Name, "copy PKI")

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
			err := copyFileOverNode(masterNode, node, "/etc/kubernetes/pki/"+file, nil)
			if err != nil {
				errChan <- err
			}
		}
	}

	for i, command := range commands {
		cluster.coordinator.AddEvent(node.Name, command.eventName)
		_, err := runCmd(node, command.command)
		if err != nil {
			errChan <- err
		}

		if numMaster > 0 && i > 0 {
			break
		}
	}

	if !cluster.HaEnabled {
		cluster.coordinator.AddEvent(node.Name, pkg.CompletedEvent)
	}

	trueChan <- true
}

func (cluster *Cluster) InstallEtcdNodes(nodes []Node) error {

	commands := []SSHCommand{
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
			err := writeNodeFile(node, "/etc/systemd/system/etcd.service", etcdSystemdService, false)
			if err != nil {
				errChan <- err
			}

			// install etcd
			for _, command := range commands {
				cluster.coordinator.AddEvent(node.Name, command.eventName)
				_, err := runCmd(node, command.command)
				if err != nil {
					errChan <- err
				}
			}
			if cluster.IsolatedEtcd {
				cluster.coordinator.AddEvent(node.Name, pkg.CompletedEvent)
			} else {
				cluster.coordinator.AddEvent(node.Name, "etcd configured")
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
				output, err := runCmd(node, "kubeadm token create --print-join-command")
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
			cluster.coordinator.AddEvent(node.Name, "registering node")
			if cluster.HaEnabled {
				// joinCommand = strings.Replace(joinCommand, "https://" + masterNode.IPAddress + ":6443", "https://127.0.0.1:16443", 1)
			}
			_, err := runCmd(node, "kubeadm reset && "+joinCommand)
			if err != nil {
				return err
			}

			if cluster.HaEnabled {
				time.Sleep(10 * time.Second) // we need some time until the kubelet.conf appears

				rewriteTpl := `cat /etc/kubernetes/%s | sed -e 's/server: https\(.*\)/server: https:\/\/127.0.0.1:16443/g' > /tmp/cp && mv /tmp/cp /etc/kubernetes/%s`
				kubeConfigs := []string{"kubelet.conf", "bootstrap-kubelet.conf"}

				cluster.coordinator.AddEvent(node.Name, "rewrite kubeconfigs")
				for _, conf := range kubeConfigs {
					_, err := runCmd(node, fmt.Sprintf(rewriteTpl, conf, conf))
					if err != nil {
						return err
					}
				}
				_, err = runCmd(node, "systemctl restart docker && systemctl restart kubelet")
				if err != nil {
					return err
				}
			}

			cluster.coordinator.AddEvent(node.Name, pkg.CompletedEvent)
		}
	}

	return nil
}

func (cluster *Cluster) SetupHA() error {
	// copy pki
	masterNode, err := cluster.GetMasterNode()
	if err != nil {
		return err
	}

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	// deploy load balancer
	masterNodes := cluster.GetMasterNodes()
	err = cluster.DeployLoadBalancer(cluster.Nodes)
	if err != nil {
		return err
	}

	// set apiserver-count to 3
	for _, node := range masterNodes {
		cluster.coordinator.AddEvent(node.Name, "set api-server count")
		copyFileOverNode(node, node, "/etc/kubernetes/manifests/kube-apiserver.yaml", func(in string) string {
			return strings.Replace(in, "image: gcr.io/", "- --apiserver-count=3\n    image: gcr.io/", 1)
		})
	}

	cluster.coordinator.AddEvent(masterNode.Name, "configuring kube-proxy")
	// update config-map for kube-proxy to lb
	proxyUpdateCmd := `kubectl get -n kube-system configmap/kube-proxy -o=yaml | sed -e 's/server: https\(.*\)/server: https:\/\/127.0.0.1:16443/g' | kubectl -n kube-system apply -f -`
	runCmd(*masterNode, proxyUpdateCmd)

	// delete proxy pods
	runCmd(*masterNode, "kubectl get pods --all-namespaces | grep proxy | awk '{print$2}' | xargs kubectl -n kube-system delete pod")

	// rewrite all kubeconfigs
	rewriteTpl := `cat /etc/kubernetes/%s | sed -e 's/server: https\(.*\)/server: https:\/\/127.0.0.1:16443/g' > /tmp/cp && mv /tmp/cp /etc/kubernetes/%s`
	kubeConfigs := []string{"kubelet.conf", "controller-manager.conf", "scheduler.conf"}

	numProcs = 0
	for _, node := range masterNodes {
		numProcs++

		go func(node Node) {
			cluster.coordinator.AddEvent(node.Name, "rewrite kubeconfigs")
			for _, conf := range kubeConfigs {
				_, err := runCmd(node, fmt.Sprintf(rewriteTpl, conf, conf))
				if err != nil {
					errChan <- err
				}
			}
			_, err = runCmd(node, "systemctl restart docker && systemctl restart kubelet")
			if err != nil {
				errChan <- err
			}

			// wait for the apiserver to be back online
			cluster.coordinator.AddEvent(node.Name, "wait for apiserver")
			_, err = runCmd(node, `until $(kubectl get node > /dev/null 2>/dev/null ); do echo "wait.."; sleep 1; done`)
			cluster.coordinator.AddEvent(node.Name, pkg.CompletedEvent)

			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProcs)
}

func (cluster *Cluster) DeployLoadBalancer(nodes []Node) error {

	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	masterNodes := cluster.GetMasterNodes()
	masterIps := strings.Join(Nodes2IPs(masterNodes), " ")
	for _, node := range nodes {
		if !node.IsMaster && node.IsEtcd {
			continue
		}
		numProcs++
		go func(node Node) {
			cluster.coordinator.AddEvent(node.Name, "deploy load balancer")
			// delete old if exists
			_, err := runCmd(node, `docker ps | grep master-lb | awk '{print "docker stop "$1" && docker rm "$1}' | sh`)
			if err != nil {
				errChan <- err
			}
			_, err = runCmd(node, fmt.Sprintf("docker run -d --name=master-lb --restart=always -p 16443:16443 xetys/k8s-master-lb %s", masterIps))
			if err != nil {
				errChan <- err
			}

			trueChan <- true
		}(node)
	}

	return waitOrError(trueChan, errChan, &numProcs)
}

func GenerateMasterConfiguration(masterNode Node, masterNodes, etcdNodes []Node) string {
	masterConfigTpl := `apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
api:
  advertiseAddress: %s
networking:
  podSubnet: 10.244.0.0/16
apiServerCertSANs:
  - %s
  - 127.0.0.1
`
	etcdConfig := `etcd:
  endpoints:`
	masterConfig := fmt.Sprintf(masterConfigTpl, masterNode.PrivateIPAddress, masterNode.IPAddress)
	for _, node := range masterNodes {
		masterConfig = fmt.Sprintf("%s%s\n", masterConfig, "  - "+node.PrivateIPAddress)
	}

	if len(etcdNodes) > 0 {
		masterConfig = masterConfig + etcdConfig + "\n"
		for _, node := range etcdNodes {
			masterConfig = fmt.Sprintf("%s%s\n", masterConfig, "  - http://"+node.PrivateIPAddress+":2379")
		}
	}

	return masterConfig
}

func GenerateEtcdSystemdService(node Node, etcdNodes []Node) string {
	serviceTpls := `# /etc/systemd/system/etcd.service
[Unit]
Description=etcd
After=network.target wg-quick@wg0.service

[Service]
ExecStart=/opt/etcd/etcd --name %s \
  --data-dir /var/lib/etcd \
  --listen-client-urls "http://%s:2379,http://localhost:2379" \
  --advertise-client-urls "http://%s:2379" \
  --listen-peer-urls "http://%s:2380" \
  --initial-cluster "%s" \
  --initial-advertise-peer-urls "http://%s:2380" \
  --heartbeat-interval 200 \
  --election-timeout 5000
Restart=always
RestartSec=5
TimeoutStartSec=0
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
`

	var ips []string
	for _, node := range etcdNodes {
		ips = append(ips, fmt.Sprintf("%s=http://%s:2380", node.Name, node.PrivateIPAddress))
	}
	initialCluster := strings.Join(ips, ",")

	service := fmt.Sprintf(
		serviceTpls,
		node.Name,
		node.PrivateIPAddress,
		node.PrivateIPAddress,
		node.PrivateIPAddress,
		initialCluster,
		node.PrivateIPAddress,
	)

	return service
}
