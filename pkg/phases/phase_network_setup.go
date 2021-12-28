package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

// NetworkSetupPhase defines the wireguard encrypted network setup phase
type NetworkSetupPhase struct {
	clusterManager *clustermanager.Manager
}

// NewNetworkSetupPhase returns an instance of *NetworkSetupPhase
func NewNetworkSetupPhase(manager *clustermanager.Manager) Phase {
	return &NetworkSetupPhase{
		clusterManager: manager,
	}
}

// ShouldRun returns if this phase should run
func (phase *NetworkSetupPhase) ShouldRun() bool {
	return false
}

// Run runs the phase
func (phase *NetworkSetupPhase) Run() error {
	err := phase.clusterManager.SetupEncryptedNetwork()
	FatalOnError(err)

	return nil
}
