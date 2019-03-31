package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/cmd/phases"
	"github.com/xetys/hetzner-kube/pkg"
)

func init() {
	declarePhaseCommand("provision", "provisions all nodes with the current tools", func(cmd *cobra.Command, args []string) {
		provider, clusterManager, coordinator := getCommonPhaseDependencies(8, cmd, args)

		phase := phases.NewProvisionNodesPhase(clusterManager)

		err := phase.Run()
		FatalOnError(err)

		for _, node := range provider.GetAllNodes() {
			coordinator.AddEvent(node.Name, pkg.CompletedEvent)
		}

		coordinator.Wait()
	})
}
