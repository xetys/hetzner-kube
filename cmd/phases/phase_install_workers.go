package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

type InstallWorkersPhase struct {
	clusterManager *clustermanager.Manager
}

func NewInstallWorkersPhase(manager *clustermanager.Manager) Phase {
	return &InstallWorkersPhase{
		clusterManager: manager,
	}
}

func (phase *InstallWorkersPhase) ShouldRun() bool {
	return true
}

func (phase *InstallWorkersPhase) Run() error {
	nodes := phase.clusterManager.Cluster().Nodes
	return phase.clusterManager.InstallWorkers(nodes)
}
