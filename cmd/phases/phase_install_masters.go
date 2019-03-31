package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

type InstallMastersPhase struct {
	clusterManager *clustermanager.Manager
}

func NewInstallMastersPhase(manager *clustermanager.Manager) Phase {
	return &InstallMastersPhase{
		clusterManager: manager,
	}
}

func (phase *InstallMastersPhase) ShouldRun() bool {
	return true
}

func (phase *InstallMastersPhase) Run() error {
	return phase.clusterManager.InstallMasters()
}
