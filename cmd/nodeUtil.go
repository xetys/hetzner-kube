package cmd

import (
	"io/ioutil"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"strings"
	"log"
	"sync"
	"errors"
)

func (cluster *Cluster) InstallWorkers(nodes []Node) error {
	var joinCommand string
	// find master
	for _, node := range cluster.Nodes {
		if node.IsMaster {
			output, err := runCmd(node, "kubeadm token create --print-join-command")
			if err != nil {
				return err
			}
			joinCommand = output
			break
		}
	}

	// now let the nodes join

	for _, node := range nodes {
		if !node.IsMaster {
			cluster.coordinator.AddEvent(node.Name, "registering node")
			_, err := runCmd(node, "swapoff -a && "+joinCommand)
			if err != nil {
				return err
			}

			cluster.coordinator.AddEvent(node.Name, "complete!")
		}
	}

	return nil
}

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

		// create
		server, err := cluster.runCreateServer(&serverOpts)

		if err != nil {
			return nil, err
		}

		ipAddress := server.Server.PublicNet.IPv4.IP.String()
		log.Printf("Created node '%s' with IP %s", server.Server.Name, ipAddress)
		privateIpLastBlock := 10 + nodeNumber
		if !template.IsMaster {
			privateIpLastBlock += 10
		}
		privateIpAddress := fmt.Sprintf("10.0.1.%d", privateIpLastBlock)

		node := Node{
			Name:             serverOpts.Name,
			Type:             serverOpts.ServerType.Name,
			IsMaster:         template.IsMaster,
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
	var wg sync.WaitGroup
	for _, node := range cluster.Nodes {

		wg.Add(1)
		go func(node Node) {
			cluster.coordinator.AddEvent(node.Name, "install packages")
			_, err := runCmd(node, "wget -cO- https://raw.githubusercontent.com/xetys/hetzner-kube/master/install-docker-kubeadm.sh | bash -")

			if err != nil {
				log.Fatalln(err)
			}

			if node.IsMaster {
				cluster.coordinator.AddEvent(node.Name, "packages installed")
			} else {
				cluster.coordinator.AddEvent(node.Name, "waiting for master")
			}

			wg.Done()

		}(node)
	}

	wg.Wait()

	return nil
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
	for _, node := range nodes {
		cluster.coordinator.AddEvent(node.Name, "configure wireguard")
		wireGuardConf := GenerateWireguardConf(node, cluster.Nodes)
		err := writeNodeFile(node, "/etc/wireguard/wg0.conf", wireGuardConf, false)
		if err != nil {
			return err
		}

		_, err = runCmd(node, "systemctl enable wg-quick@wg0 && systemctl restart wg-quick@wg0")

		if err != nil {
			return err
		}
	}

	return nil
}

func (cluster *Cluster) runCreateServer(opts *hcloud.ServerCreateOpts) (*hcloud.ServerCreateResult, error) {

	log.Printf("creating server '%s'...", opts.Name)
	result, _, err := AppConf.Client.Server.Create(AppConf.Context, *opts)
	if err != nil {
		if err.(hcloud.Error).Code == "uniqueness_error" {
			server, _, err := AppConf.Client.Server.Get(AppConf.Context, opts.Name)

			if err != nil {
				return nil, err
			}

			log.Printf("loading server '%s'...", opts.Name)
			return &hcloud.ServerCreateResult{Server: server}, nil
		}

		return nil, err
	}

	if err := AppConf.ActionProgress(AppConf.Context, result.Action); err != nil {
		return nil, err
	}

	cluster.wait = true

	return &result, nil
}

func (cluster *Cluster) GetMasterNode() (node *Node, err error) {

	for _, node := range cluster.Nodes {
		if node.IsMaster {
			return &node, nil
		}
	}

	return nil, errors.New("no master node found")
}

func (cluster *Cluster) CreateMasterNodes(sshKeyName string, masterServerType string, datacenters []string, count int) error {
	template := Node{SSHKeyName: sshKeyName, IsMaster: true, Type: masterServerType}
	log.Println("creating master nodes...")
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

func (cluster *Cluster) InstallMaster() error {

	commands := []SSHCommand{
		{"disable swap", "swapoff -a"},
		{"kubeadm init", "kubeadm reset && kubeadm init --config /root/master-config.yaml"},
		{"configure kubectl", "mkdir -p $HOME/.kube && cp -i /etc/kubernetes/admin.conf $HOME/.kube/config && chown $(id -u):$(id -g) $HOME/.kube/config"},
		{"install flannel", "kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/v0.9.1/Documentation/kube-flannel.yml"},
		{"configure flannel", "kubectl -n kube-system patch ds kube-flannel-ds --type json -p '[{\"op\":\"add\",\"path\":\"/spec/template/spec/tolerations/-\",\"value\":{\"key\":\"node.cloudprovider.kubernetes.io/uninitialized\",\"value\":\"true\",\"effect\":\"NoSchedule\"}}]'"},
		{"install hcloud integration", fmt.Sprintf("kubectl -n kube-system create secret generic hcloud --from-literal=token=%s", AppConf.CurrentContext.Token)},
		{"deploy cloud controller manager", "kubectl apply -f  https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/v1.0.0.yaml"},
	}

	for _, node := range cluster.Nodes {
		if node.IsMaster {
			if len(cluster.Nodes) == 1 {
				commands = append(commands, SSHCommand{"taint master", "kubectl taint nodes --all node-role.kubernetes.io/master-"})
			}

			// create master-configuration
			masterConfig := GenerateMasterConfiguration(node, nil)
			if err := writeNodeFile(node, "/root/master-config.yaml", masterConfig, false); err != nil {
				return err
			}

			for _, command := range commands {
				cluster.coordinator.AddEvent(node.Name, command.eventName)
				_, err := runCmd(node, command.command)
				if err != nil {
					return err
				}
			}

			cluster.coordinator.AddEvent(node.Name, "complete!")
			break
		}
	}

	return nil
}

func GenerateMasterConfiguration(masterNode Node, etcdNodes []Node) string {
	masterConfigTpl := `apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
api:
  advertiseAddress: %s
networking:
  podSubnet: 10.244.0.0/16
apiServerCertSANs:
  - %s
  - 10.0.1.11
  - 127.0.0.1
`
	etcdConfig := `etcd:
  endpoints:`
	masterConfig := fmt.Sprintf(masterConfigTpl, masterNode.PrivateIPAddress, masterNode.IPAddress)

	if len(etcdNodes) > 0 {
		masterConfig = masterConfig + etcdConfig + "\n"
		for _, node := range etcdNodes {
			masterConfig = fmt.Sprintf("%s%s\n", masterConfig, "  - http://"+node.PrivateIPAddress+":2379")
		}
	}

	return masterConfig
}
