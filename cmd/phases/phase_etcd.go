package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

type EtcdSetupPhase struct {
	clusterManager *clustermanager.Manager
	provider       clustermanager.ClusterProvider
}

func NewEtcdSetupPhase(manager *clustermanager.Manager, provider clustermanager.ClusterProvider) Phase {
	return &EtcdSetupPhase{
		clusterManager: manager,
		provider:       provider,
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

	err := phase.clusterManager.InstallEtcdNodes(etcdNodes)
	if err != nil {
		return err
	}

	return nil
}
