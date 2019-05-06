package phases

import (
	"fmt"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"log"
)

// ProvisionNodesPhase defines the phase which install all the tools for each node
type ProvisionNodesPhase struct {
	clusterManager *clustermanager.Manager
}

// NewProvisionNodesPhase returns an instance of *ProvisionNodesPhase
func NewProvisionNodesPhase(manager *clustermanager.Manager) Phase {
	return &ProvisionNodesPhase{
		clusterManager: manager,
	}
}

// ShouldRun returns if this phase should run
func (phase *ProvisionNodesPhase) ShouldRun() bool {
	return true
}

// Run runs the phase
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
