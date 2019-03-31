package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

type SetupHighAvailabilityPhase struct {
	clusterManager *clustermanager.Manager
}

func NewSetupHighAvailabilityPhase(manager *clustermanager.Manager) Phase {
	return &SetupHighAvailabilityPhase{
		clusterManager: manager,
	}
}

func (phase *SetupHighAvailabilityPhase) ShouldRun() bool {
	return phase.clusterManager.Cluster().HaEnabled
}

func (phase *SetupHighAvailabilityPhase) Run() error {
	return phase.clusterManager.SetupHA()
}
