package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

type EtcdSetupPhaseOptions struct {
	KeepData bool
}

type EtcdSetupPhase struct {
	clusterManager *clustermanager.Manager
	provider       clustermanager.ClusterProvider
	options        EtcdSetupPhaseOptions
}

func NewEtcdSetupPhase(manager *clustermanager.Manager, provider clustermanager.ClusterProvider, options EtcdSetupPhaseOptions) Phase {
	return &EtcdSetupPhase{
		clusterManager: manager,
		provider:       provider,
		options:        options,
	}
}

func (phase *EtcdSetupPhase) ShouldRun() bool {
	return phase.clusterManager.Cluster().HaEnabled
}

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
