package cmd

import (
	"context"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

type HetznerContext struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

// deprecated
type SSHKey struct {
	Name           string `json:"name"`
	PrivateKeyPath string `json:"private_key_path"`
	PublicKeyPath  string `json:"public_key_path"`
}

type HetznerConfig struct {
	ActiveContextName string                   `json:"active_context_name"`
	Contexts          []HetznerContext         `json:"contexts"`
	SSHKeys           []clustermanager.SSHKey  `json:"ssh_keys"`
	Clusters          []clustermanager.Cluster `json:"clusters"`
}

type AppConfig struct {
	Client         *hcloud.Client
	Context        context.Context
	CurrentContext *HetznerContext
	Config         *HetznerConfig
	SSHClient      clustermanager.NodeCommunicator
}
