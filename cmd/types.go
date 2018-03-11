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
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	IsMaster         bool      `json:"is_master"`
	IsEtcd           bool      `json:"is_etcd"`
	IPAddress        string    `json:"ip_address"`
	PrivateIPAddress string    `json:"private_ip_address"`
	SSHKeyName       string    `json:"ssh_key_name"`
	WireGuardKeyPair WgKeyPair `json:"wire_guard_key_pair"`
}

type Cluster struct {
	Name          string                   `json:"name"`
	Nodes         []Node                   `json:"nodes"`
	SelfHosted    bool                     `json:"self_hosted"`
	coordinator   *pkg.UiProgressCoordinator
	wait          bool
	CloudInitFile string                   `json:"cloud_init_file"`
	HaEnabled     bool                     `json:"ha_enabled"`
	IsolatedEtcd  bool                     `json:"isolated_etcd"`
}

type SSHCommand struct {
	eventName string
	command   string
}


type SSHClient interface {
	RunCmd(node *Node, cmd string) (string, error)
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
	SSHClient      SSHClient
}
