package hetzner

import (
	"context"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

//ProviderAndManager get the provider and the manager for the cluster
func ProviderAndManager(context context.Context, cluster clustermanager.Cluster, client *hcloud.Client, nc clustermanager.NodeCommunicator, eventService clustermanager.EventService, token string) (*Provider, *clustermanager.Manager) {
	provider := NewHetznerProvider(context, client, cluster, token)

	manager := clustermanager.NewClusterManagerFromCluster(cluster, provider, nc, eventService)

	return provider, manager
}
