package phases

import (
	"fmt"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"log"
)

type ProvisionNodesPhase struct {
	clusterManager *clustermanager.Manager
}

func NewProvisionNodesPhase(manager *clustermanager.Manager) Phase {
	return &ProvisionNodesPhase{
		clusterManager: manager,
	}
}

func (phase *ProvisionNodesPhase) ShouldRun() bool {
	return true
}

func (phase *ProvisionNodesPhase) Run() error {
	cluster := phase.clusterManager.Cluster()

	tries := 0
	for err := phase.clusterManager.ProvisionNodes(cluster.Nodes); err != nil; {
		if tries < 3 {
			fmt.Print(err)
			tries++
		} else {
			log.Fatal(err)
		}
	}

	return nil
}
