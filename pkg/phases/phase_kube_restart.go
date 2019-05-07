package phases

import (
	"fmt"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

// KubeRestartPhase defines a simple phase which restarts docker and kubelet
type KubeRestartPhase struct {
	provider clustermanager.ClusterProvider
	ssh      clustermanager.NodeCommunicator
}

// NewKubeRestartPhase returns an instance of *KubeRestartPhase
func NewKubeRestartPhase(provider clustermanager.ClusterProvider, ssh clustermanager.NodeCommunicator) Phase {
	return &KubeRestartPhase{
		provider: provider,
		ssh:      ssh,
	}
}

// ShouldRun returns if this phase should run
func (phase *KubeRestartPhase) ShouldRun() bool {
	return true
}

// Run runs the phase
func (phase *KubeRestartPhase) Run() error {
	fmt.Println("restarting")
	errChan := make(chan error)
	trueChan := make(chan bool)
	numProcs := 0
	for _, node := range phase.provider.GetAllNodes() {
		numProcs++
		go func(node clustermanager.Node) {
			fmt.Printf("restarting docker+kubelet on node '%s'\n", node.Name)
			_, err := phase.ssh.RunCmd(node, "systemctl restart docker && systemctl restart kubelet")

			if err != nil {
				errChan <- err
			}

			fmt.Printf("restarted docker+kubelet on node '%s'\n", node.Name)
			trueChan <- true
		}(node)
	}

	for numProcs > 0 {
		select {
		case err := <-errChan:
			return err
		case <-trueChan:
			numProcs--
		}
	}

	return nil
}
