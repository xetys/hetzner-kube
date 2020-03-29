package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

// InstallMastersPhaseOptions contains option for the control plane node setup
type InstallMastersPhaseOptions struct {
	KeepCaCerts  bool
	KeepAllCerts bool
}

// InstallMastersPhase defines the phase for installing kubernetes master nodes
type InstallMastersPhase struct {
	clusterManager *clustermanager.Manager
	options        InstallMastersPhaseOptions
}

// NewInstallMastersPhase returns a new instance of *InstallMastersPhase
func NewInstallMastersPhase(manager *clustermanager.Manager, options InstallMastersPhaseOptions) Phase {
	return &InstallMastersPhase{
		clusterManager: manager,
		options:        options,
	}
}

// ShouldRun returns if this phase should run
func (phase *InstallMastersPhase) ShouldRun() bool {
	return true
}

// Run runs the phase
func (phase *InstallMastersPhase) Run() error {
	keepCerts := clustermanager.NONE

	if phase.options.KeepAllCerts {
		keepCerts = clustermanager.ALL
	} else if phase.options.KeepCaCerts {
		keepCerts = clustermanager.CA
	}

	return phase.clusterManager.InstallMasters(keepCerts)
}
