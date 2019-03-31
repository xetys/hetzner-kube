package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

type InstallMastersPhaseOptions struct {
	KeepCaCerts  bool
	KeepAllCerts bool
}

type InstallMastersPhase struct {
	clusterManager *clustermanager.Manager
	options        InstallMastersPhaseOptions
}

func NewInstallMastersPhase(manager *clustermanager.Manager, options InstallMastersPhaseOptions) Phase {
	return &InstallMastersPhase{
		clusterManager: manager,
		options:        options,
	}
}

func (phase *InstallMastersPhase) ShouldRun() bool {
	return true
}

func (phase *InstallMastersPhase) Run() error {
	keepCerts := clustermanager.NONE
	if phase.options.KeepAllCerts {
		keepCerts = clustermanager.ALL
	} else if phase.options.KeepCaCerts {
		keepCerts = clustermanager.CA
	}
	return phase.clusterManager.InstallMasters(keepCerts)
}
