package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/cmd/phases"
	"github.com/xetys/hetzner-kube/pkg"
)

func init() {
	declarePhaseCommand("etcd", "setups a etcd cluster", func(cmd *cobra.Command, args []string) {
		provider, clusterManager, coordinator := getCommonPhaseDependencies(6, cmd, args)

		phase := phases.NewEtcdSetupPhase(clusterManager, provider)

		if phase.ShouldRun() {
			err := phase.Run()
			FatalOnError(err)
		}

		for _, node := range provider.GetAllNodes() {
			coordinator.AddEvent(node.Name, pkg.CompletedEvent)
		}

		coordinator.Wait()
	})
}
