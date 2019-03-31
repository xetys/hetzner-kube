package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/cmd/phases"
	"github.com/xetys/hetzner-kube/pkg"
)

func init() {
	declarePhaseCommand("network-setup", "setups wireguard encrypted network", func(cmd *cobra.Command, args []string) {
		provider, clusterManager, coordinator := getCommonPhaseDependencies(2, cmd, args)
		phase := phases.NewNetworkSetupPhase(clusterManager)

		err := phase.Run()
		FatalOnError(err)

		for _, node := range provider.GetAllNodes() {
			coordinator.AddEvent(node.Name, pkg.CompletedEvent)
		}

		coordinator.Wait()
	})
}
