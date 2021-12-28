package phases

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

// DeployLoadBalancerPhase defines the phase where the HA proxy load balancer is deployed
type DeployLoadBalancerPhase struct {
	clusterManager *clustermanager.Manager
}

// NewDeployLoadBalancerPhase returns an instance of *NewDeployLoadBalancerPhase
func NewDeployLoadBalancerPhase(manager *clustermanager.Manager) Phase {
	return &DeployLoadBalancerPhase{
		clusterManager: manager,
	}
}

// ShouldRun returns if this phase should run
func (phase *DeployLoadBalancerPhase) ShouldRun() bool {
	return phase.clusterManager.Cluster().HaEnabled
}

// Run runs the phase
func (phase *DeployLoadBalancerPhase) Run() error {
	return phase.clusterManager.SetupHAProxyForAllNodes()
}
