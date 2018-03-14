package hetzner

import (
	"context"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"time"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

func ProviderAndManager(cluster clustermanager.Cluster, client *hcloud.Client, context context.Context, nc clustermanager.NodeCommunicator, eventService clustermanager.EventService) (*Provider, *clustermanager.Manager) {
	provider := NewHetznerProvider(cluster.Name, client, context)
	provider.SetNodes(cluster.Nodes)
	manager := clustermanager.NewClusterManagerFromCluster(cluster, provider, nc, eventService)

	return provider, manager
}

func waitAction(ctx context.Context, client *hcloud.Client, action *hcloud.Action) (<-chan error, <-chan int) {
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
				break
			}

			action, _, err := client.Action.GetByID(ctx, action.ID)
			if err != nil {
				errCh <- ctx.Err()
				return
			}

			switch action.Status {
			case hcloud.ActionStatusRunning:
				sendProgress(action.Progress)
				break
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
