package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

// SetupHighAvailabilityPhase defines the phase where the components are configured for a HA control plane
type SetupHighAvailabilityPhase struct {
	clusterManager *clustermanager.Manager
}

// NewSetupHighAvailabilityPhase returns an instance of *NewSetupHighAvailabilityPhase
func NewSetupHighAvailabilityPhase(manager *clustermanager.Manager) Phase {
	return &SetupHighAvailabilityPhase{
		clusterManager: manager,
	}
}

// ShouldRun returns if this phase should run
func (phase *SetupHighAvailabilityPhase) ShouldRun() bool {
	return phase.clusterManager.Cluster().HaEnabled
}

// Run runs the phase
func (phase *SetupHighAvailabilityPhase) Run() error {
	return phase.clusterManager.SetupHA()
}
