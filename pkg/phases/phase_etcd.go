package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

// EtcdSetupPhaseOptions contains options for etcd setup
type EtcdSetupPhaseOptions struct {
	KeepData bool
}

// EtcdSetupPhase defines the phase for setting up etcd clusters
type EtcdSetupPhase struct {
	clusterManager *clustermanager.Manager
	provider       clustermanager.ClusterProvider
	options        EtcdSetupPhaseOptions
}

// NewEtcdSetupPhase returns an *EtcdSetupPhase instance
func NewEtcdSetupPhase(manager *clustermanager.Manager, provider clustermanager.ClusterProvider, options EtcdSetupPhaseOptions) Phase {
	return &EtcdSetupPhase{
		clusterManager: manager,
		provider:       provider,
		options:        options,
	}
}

// ShouldRun returns if this phase should run
func (phase *EtcdSetupPhase) ShouldRun() bool {
	return false //phase.clusterManager.Cluster().HaEnabled
}

// Run runs the phase
func (phase *EtcdSetupPhase) Run() error {
	var etcdNodes []clustermanager.Node
	cluster := phase.clusterManager.Cluster()

	if cluster.IsolatedEtcd {
		etcdNodes = phase.provider.GetEtcdNodes()
	} else {
		etcdNodes = phase.provider.GetMasterNodes()
	}

	err := phase.clusterManager.InstallEtcdNodes(etcdNodes, phase.options.KeepData)
	if err != nil {
		return err
	}

	return nil
}
