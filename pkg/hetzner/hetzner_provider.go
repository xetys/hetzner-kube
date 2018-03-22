package hetzner

import (
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"fmt"
	"io/ioutil"
	"strings"
	"context"
	"log"
	"github.com/go-kit/kit/log/term"
	"os"
	"github.com/gosuri/uiprogress"
	"errors"
)

type Provider struct {
	client *hcloud.Client
	context context.Context
	nodes []clustermanager.Node
	clusterName string
	cloudInitFile string
	wait bool
	token string
}

func NewHetznerProvider(clusterName string, client *hcloud.Client, context context.Context, token string) *Provider {

	return &Provider{client: client, context:context,clusterName:clusterName, token: token}
}

func (provider *Provider) SetCloudInitFile(cloudInitFile string)  {
	provider.cloudInitFile = cloudInitFile
}

func (provider *Provider) CreateNodes(suffix string, template clustermanager.Node, datacenters []string, count int, offset int) ([]clustermanager.Node, error) {
	sshKey, _, err := provider.client.SSHKey.Get(provider.context, template.SSHKeyName)

	if err != nil {
		return nil, err
	}

	serverNameTemplate := fmt.Sprintf("%s-%s-@idx", provider.clusterName, suffix)
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
		var serverOpts hcloud.ServerCreateOpts
		serverOpts = serverOptsTemplate
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
		privateIpLastBlock := nodeNumber
		if !template.IsEtcd {
			privateIpLastBlock += 10
			if !template.IsMaster {
				privateIpLastBlock += 10
			}
		}
		privateIpAddress := fmt.Sprintf("10.0.1.%d", privateIpLastBlock)

		node := clustermanager.Node{
			Name:             serverOpts.Name,
			Type:             serverOpts.ServerType.Name,
			IsMaster:         template.IsMaster,
			IsEtcd:           template.IsEtcd,
			IPAddress:        ipAddress,
			PrivateIPAddress: privateIpAddress,
			SSHKeyName:       template.SSHKeyName,
		}
		nodes = append(nodes, node)
		provider.nodes = append(provider.nodes, node)
	}

	return nodes, nil
}


func (provider *Provider) CreateEtcdNodes(sshKeyName string, masterServerType string, datacenters []string, count int) error {
	template := clustermanager.Node{SSHKeyName: sshKeyName, IsEtcd: true, Type: masterServerType}
	_, err := provider.CreateNodes("etcd", template, datacenters, count, 0)
	return err
}

func (provider *Provider) CreateMasterNodes(sshKeyName string, masterServerType string, datacenters []string, count int, isEtcd bool) error {
	template := clustermanager.Node{SSHKeyName: sshKeyName, IsMaster: true, Type: masterServerType, IsEtcd: isEtcd}
	_, err := provider.CreateNodes("master", template, datacenters, count, 0)
	return err
}

func (provider *Provider) CreateWorkerNodes(sshKeyName string, workerServerType string, datacenters []string, count int, offset int) ([]clustermanager.Node, error) {
	template := clustermanager.Node{SSHKeyName: sshKeyName, IsMaster: false, Type: workerServerType}
	nodes, err := provider.CreateNodes("worker", template, datacenters, count, offset)
	return nodes, err
}

func (provider *Provider) GetAllNodes() []clustermanager.Node {

	return provider.nodes
}

func (provider *Provider) SetNodes(nodes []clustermanager.Node) {
	provider.nodes = nodes
}

func (provider *Provider) GetMasterNodes() []clustermanager.Node {
	nodes := []clustermanager.Node{}
	for _, node := range provider.nodes {
		if node.IsMaster {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func (provider *Provider) GetEtcdNodes() []clustermanager.Node {

	nodes := []clustermanager.Node{}
	for _, node := range provider.nodes {
		if node.IsEtcd {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func (provider *Provider) GetWorkerNodes() []clustermanager.Node {
	nodes := []clustermanager.Node{}
	for _, node := range provider.nodes {
		if !node.IsMaster && !node.IsEtcd {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func (provider *Provider) GetMasterNode() (*clustermanager.Node, error) {
	for _, node := range provider.nodes {
		if node.IsMaster {
			return &node, nil
		}
	}

	return nil, errors.New("no master node found")
}

func (provider *Provider) GetCluster() clustermanager.Cluster {

	return clustermanager.Cluster{
		Name: provider.clusterName,
		Nodes: provider.nodes,
	}
}

func (provider *Provider) GetAdditionalMasterInstallCommands() []clustermanager.NodeCommand {

	return []clustermanager.NodeCommand{
		{"configure flannel", "kubectl -n kube-system patch ds kube-flannel-ds --type json -p '[{\"op\":\"add\",\"path\":\"/spec/template/spec/tolerations/-\",\"value\":{\"key\":\"node.cloudprovider.kubernetes.io/uninitialized\",\"value\":\"true\",\"effect\":\"NoSchedule\"}}]'"},
		{"install hcloud integration", fmt.Sprintf("kubectl -n kube-system create secret generic hcloud --from-literal=token=%s", provider.token)},
		{"deploy cloud controller manager", "kubectl apply -f  https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/v1.0.0.yaml"},
	}
}

func (provider *Provider) MustWait() bool {
	return provider.wait
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
	} else {
		log.Printf("loading server '%s'...", opts.Name)
		return &hcloud.ServerCreateResult{Server: server}, nil
	}
}

func (provider *Provider) actionProgress(action *hcloud.Action) error {
	errCh, progressCh := waitAction(provider.context, provider.client, action)

	if term.IsTerminal(os.Stdout) {
		progress := uiprogress.New()

		progress.Start()
		bar := progress.AddBar(100).AppendCompleted().PrependElapsed()
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
