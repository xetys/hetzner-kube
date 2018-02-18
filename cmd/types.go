package cmd

import (
	"context"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/xetys/hetzner-kube/pkg"
)

type HetznerContext struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

type SSHKey struct {
	Name           string `json:"name"`
	PrivateKeyPath string `json:"private_key_path"`
	PublicKeyPath  string `json:"public_key_path"`
}

type Node struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsMaster   bool   `json:"is_master"`
	IPAddress  string `json:"ip_address"`
	SSHKeyName string `json:"ssh_key_name"`
}

type Cluster struct {
	Name          string                   `json:"name"`
	Nodes         []Node                   `json:"nodes"`
	SelfHosted    bool                     `json:"self_hosted"`
	coordinator   *pkg.ProgressCoordinator `json:"-"`
	wait          bool                     `json:"-"`
	CloudInitFile string                   `json:cloud_init_file`
}

type SSHCommand struct {
	eventName string
	command   string
}

type ClusterManager interface {
	CreateMasterNodes(template Node, count int) error
	CreateWorkerNodes(template Node, count int) error
	ProvisionNodes() error
	InstallMaster()
	InstallWorkers()
	GetKubeconfig()
}

type SSHClient interface {
	RunCmd(node *Node, cmd string)
}

type HetznerConfig struct {
	ActiveContextName string           `json:"active_context_name"`
	Contexts          []HetznerContext `json:"contexts"`
	SSHKeys           []SSHKey         `json:"ssh_keys"`
	Clusters          []Cluster        `json:"clusters"`
}

type AppConfig struct {
	Client         *hcloud.Client
	Context        context.Context
	CurrentContext *HetznerContext
	Config         *HetznerConfig
}
