package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

type NetworkSetupPhase struct {
	clusterManager *clustermanager.Manager
}

func NewNetworkSetupPhase(manager *clustermanager.Manager) Phase {
	return &NetworkSetupPhase{
		clusterManager: manager,
	}
}

func (phase *NetworkSetupPhase) ShouldRun() bool {
	return true
}

func (phase *NetworkSetupPhase) Run() error {
	err := phase.clusterManager.SetupEncryptedNetwork()
	FatalOnError(err)
	//cluster := phase.clusterManager.Cluster()
	//saveCluster(&cluster)

	return nil
}
