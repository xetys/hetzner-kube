package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

// InstallWorkersPhase defines the phase which installs worker nodes
type InstallWorkersPhase struct {
	clusterManager *clustermanager.Manager
}

// NewInstallWorkersPhase returns an instance *InstallWorkersPhase
func NewInstallWorkersPhase(manager *clustermanager.Manager) Phase {
	return &InstallWorkersPhase{
		clusterManager: manager,
	}
}

// ShouldRun returns if this phase should run
func (phase *InstallWorkersPhase) ShouldRun() bool {
	return true
}

// Run runs the phase
func (phase *InstallWorkersPhase) Run() error {
	nodes := phase.clusterManager.Cluster().Nodes
	return phase.clusterManager.InstallWorkers(nodes)
}
