package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg"
	phases "github.com/xetys/hetzner-kube/pkg/phases"
)

var networkSetupPhaseCommand = &cobra.Command{
	Use:     "network-setup <CLUSTER_NAME>",
	Short:   "setups wireguard encrypted network",
	Args:    cobra.ExactArgs(1),
	PreRunE: validateClusterInArgumentExists,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, clusterManager, coordinator := getCommonPhaseDependencies(2, cmd, args)
		phase := phases.NewNetworkSetupPhase(clusterManager)

		err := phase.Run()
		if err != nil {
			return err
		}

		for _, node := range provider.GetAllNodes() {
			coordinator.AddEvent(node.Name, pkg.CompletedEvent)
		}

		coordinator.Wait()
		return nil
	},
}

func init() {
	phaseCommand.AddCommand(networkSetupPhaseCommand)
}
