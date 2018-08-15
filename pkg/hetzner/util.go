package hetzner

import (
	"context"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

//ProviderAndManager get the provider and the manager for the cluster
func ProviderAndManager(context context.Context, cluster clustermanager.Cluster, client *hcloud.Client, nc clustermanager.NodeCommunicator, eventService clustermanager.EventService, token string) (*Provider, *clustermanager.Manager) {
	provider := NewHetznerProvider(context, client, token)
	provider.InitCluster(cluster.Name, cluster.NodeCIDR)
	provider.SetNodes(cluster.Nodes)
	manager := clustermanager.NewClusterManagerFromCluster(cluster, provider, nc, eventService)

	return provider, manager
}

// WaitAction is an helper function used to wait for an action
func WaitAction(ctx context.Context, client *hcloud.Client, action *hcloud.Action) (<-chan error, <-chan int) {
	errCh := make(chan error, 1)
	progressCh := make(chan int)

	go func() {
		defer close(errCh)
		defer close(progressCh)

		ticker := time.NewTicker(100 * time.Millisecond)

		sendProgress := func(p int) {
			select {
			case progressCh <- p:
				break
			default:
				break
			}
		}

		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case <-ticker.C:
			}

			action, _, err := client.Action.GetByID(ctx, action.ID)
			if err != nil {
				errCh <- ctx.Err()
				return
			}

			switch action.Status {
			case hcloud.ActionStatusRunning:
				sendProgress(action.Progress)

			case hcloud.ActionStatusSuccess:
				sendProgress(100)
				errCh <- nil
				return
			case hcloud.ActionStatusError:
				errCh <- action.Error()
				return
			}
		}
	}()

	return errCh, progressCh
}
