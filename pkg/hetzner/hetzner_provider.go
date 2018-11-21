package hetzner

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/go-kit/kit/log/term"
	"github.com/gosuri/uiprogress"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

//Provider contains provider information
type Provider struct {
	client        *hcloud.Client
	context       context.Context
	nodes         []clustermanager.Node
	clusterName   string
	cloudInitFile string
	wait          bool
	token         string
	nodeCidr      string
}

// NewHetznerProvider returns an instance of hetzner.Provider
func NewHetznerProvider(context context.Context, client *hcloud.Client, cluster clustermanager.Cluster, token string) *Provider {
	return &Provider{
		client:        client,
		context:       context,
		token:         token,
		nodeCidr:      cluster.NodeCIDR,
		clusterName:   cluster.Name,
		cloudInitFile: cluster.CloudInitFile,
		nodes:         cluster.Nodes,
	}
}

// CreateNodes creates hetzner nodes
func (provider *Provider) CreateNodes(template clustermanager.Node, datacenters []string, count int, offset int) ([]clustermanager.Node, error) {
	sshKey, _, err := provider.client.SSHKey.Get(provider.context, template.SSHKeyName)

	if err != nil {
		return nil, err
	}

	if sshKey == nil {
		return nil, fmt.Errorf("we got some problem with the SSH-Key '%s', chances are you are in the wrong context", template.SSHKeyName)
	}

	serverNameTemplate := fmt.Sprintf("%s-%s-@idx", provider.clusterName, template.Group)
	serverOptsTemplate := hcloud.ServerCreateOpts{
		Name: serverNameTemplate,
		ServerType: &hcloud.ServerType{
			Name: template.Type,
		},
		Image: &hcloud.Image{
			Name: "ubuntu-16.04",
		},
	}

	if len(provider.cloudInitFile) > 0 {
		buf, err := ioutil.ReadFile(provider.cloudInitFile)
		if err == nil {
			serverOptsTemplate.UserData = string(buf)
		}

	}

	serverOptsTemplate.SSHKeys = append(serverOptsTemplate.SSHKeys, sshKey)

	datacentersCount := len(datacenters)

	var nodes []clustermanager.Node
	for i := 1; i <= count; i++ {
		serverOpts := serverOptsTemplate
		nodeNumber := i + offset
		serverOpts.Name = strings.Replace(serverNameTemplate, "@idx", fmt.Sprintf("%.02d", nodeNumber), 1)
		serverOpts.Datacenter = &hcloud.Datacenter{
			Name: datacenters[i%datacentersCount],
		}

		// create
		server, err := provider.runCreateServer(&serverOpts)

		if err != nil {
			return nil, err
		}

		ipAddress := server.Server.PublicNet.IPv4.IP.String()
		log.Printf("Created node '%s' with IP %s", server.Server.Name, ipAddress)

		// render private IP address
		privateIPLastBlock := nodeNumber
		if !template.IsEtcd {
			privateIPLastBlock += 10
			if !template.IsMaster {
				privateIPLastBlock += 10
			}
		}
		cidrPrefix, err := clustermanager.PrivateIPPrefix(provider.nodeCidr)
		if err != nil {
			return nil, err
		}

		privateIPAddress := fmt.Sprintf("%s.%d", cidrPrefix, privateIPLastBlock)

		node := clustermanager.Node{
			Name:             serverOpts.Name,
			Type:             serverOpts.ServerType.Name,
			IsMaster:         template.IsMaster,
			IsEtcd:           template.IsEtcd,
			IPAddress:        ipAddress,
			PrivateIPAddress: privateIPAddress,
			SSHKeyName:       template.SSHKeyName,
		}
		nodes = append(nodes, node)
		provider.nodes = append(provider.nodes, node)
	}

	return nodes, nil
}

// CreateEtcdNodes creates nodes with type 'etcd'
func (provider *Provider) CreateEtcdNodes(sshKeyName string, masterServerType string, datacenters []string, count int) ([]clustermanager.Node, error) {
	template := clustermanager.Node{SSHKeyName: sshKeyName, IsEtcd: true, Type: masterServerType, Group: "etcd"}
	return provider.CreateNodes(template, datacenters, count, 0)
}

// CreateMasterNodes creates nodes with type 'master'
func (provider *Provider) CreateMasterNodes(sshKeyName string, masterServerType string, datacenters []string, count int, isEtcd bool) ([]clustermanager.Node, error) {
	template := clustermanager.Node{SSHKeyName: sshKeyName, IsMaster: true, Type: masterServerType, IsEtcd: isEtcd, Group: "master"}
	return provider.CreateNodes(template, datacenters, count, 0)
}

// CreateWorkerNodes create new worker node on provider
func (provider *Provider) CreateWorkerNodes(sshKeyName string, workerServerType string, datacenters []string, count int, offset int) ([]clustermanager.Node, error) {
	template := clustermanager.Node{SSHKeyName: sshKeyName, IsMaster: false, Type: workerServerType, Group: "worker"}
	return provider.CreateNodes(template, datacenters, count, offset)
}

// GetAllNodes retrieves all nodes
func (provider *Provider) GetAllNodes() []clustermanager.Node {
	return provider.nodes
}

// SetNodes set list of cluster nodes for this provider
func (provider *Provider) SetNodes(nodes []clustermanager.Node) {
	provider.nodes = nodes
}

// GetMasterNodes returns master nodes only
func (provider *Provider) GetMasterNodes() []clustermanager.Node {
	return provider.filterNodes(func(node clustermanager.Node) bool {
		return node.IsMaster
	})
}

// GetEtcdNodes returns etcd nodes only
func (provider *Provider) GetEtcdNodes() []clustermanager.Node {
	return provider.filterNodes(func(node clustermanager.Node) bool {
		return node.IsEtcd
	})
}

// GetWorkerNodes returns worker nodes only
func (provider *Provider) GetWorkerNodes() []clustermanager.Node {
	return provider.filterNodes(func(node clustermanager.Node) bool {
		return !node.IsMaster && !node.IsEtcd
	})
}

// GetMasterNode returns the first master node or fail, if no master nodes are found
func (provider *Provider) GetMasterNode() (*clustermanager.Node, error) {
	nodes := provider.GetMasterNodes()
	if len(nodes) == 0 {
		return nil, errors.New("no master node found")
	}

	return &nodes[0], nil
}

// GetCluster returns a template for Cluster
func (provider *Provider) GetCluster() clustermanager.Cluster {
	return clustermanager.Cluster{
		Name:          provider.clusterName,
		Nodes:         provider.nodes,
		CloudInitFile: provider.cloudInitFile,
		NodeCIDR:      provider.nodeCidr,
	}
}

// GetAdditionalMasterInstallCommands return the list of node command to execute on the cluster
func (provider *Provider) GetAdditionalMasterInstallCommands() []clustermanager.NodeCommand {

	return []clustermanager.NodeCommand{}
}

// GetNodeCidr returns the CIDR to use for nodes in cluster
func (provider *Provider) GetNodeCidr() string {
	return provider.nodeCidr
}

// MustWait returns true, if we have to wait after creation for some time
func (provider *Provider) MustWait() bool {
	return provider.wait
}

// Token returns the hcloud token
func (provider *Provider) Token() string {
	return provider.token
}

type nodeFilter func(clustermanager.Node) bool

func (provider *Provider) filterNodes(filter nodeFilter) []clustermanager.Node {
	nodes := []clustermanager.Node{}
	for _, node := range provider.nodes {
		if filter(node) {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func (provider *Provider) runCreateServer(opts *hcloud.ServerCreateOpts) (*hcloud.ServerCreateResult, error) {
	log.Printf("creating server '%s'...", opts.Name)
	server, _, err := provider.client.Server.GetByName(provider.context, opts.Name)
	if err != nil {
		return nil, err
	}
	if server == nil {
		result, _, err := provider.client.Server.Create(provider.context, *opts)
		if err != nil {
			if err.(hcloud.Error).Code == "uniqueness_error" {
				server, _, err := provider.client.Server.Get(provider.context, opts.Name)

				if err != nil {
					return nil, err
				}

				return &hcloud.ServerCreateResult{Server: server}, nil
			}

			return nil, err
		}

		if err := provider.actionProgress(result.Action); err != nil {
			return nil, err
		}

		provider.wait = true

		return &result, nil
	}

	log.Printf("loading server '%s'...", opts.Name)
	return &hcloud.ServerCreateResult{Server: server}, nil
}

func (provider *Provider) actionProgress(action *hcloud.Action) error {
	progressCh, errCh := provider.client.Action.WatchProgress(provider.context, action)

	if term.IsTerminal(os.Stdout) {
		progress := uiprogress.New()

		progress.Start()
		bar := progress.AddBar(100).AppendCompleted().PrependElapsed()
		bar.Width = 40
		bar.Empty = ' '

		for {
			select {
			case err := <-errCh:
				if err == nil {
					bar.Set(100)
				}
				progress.Stop()
				return err
			case p := <-progressCh:
				bar.Set(p)
			}
		}
	} else {
		return <-errCh
	}
}
