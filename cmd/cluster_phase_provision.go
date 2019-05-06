package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg"
	phases "github.com/xetys/hetzner-kube/pkg/phases"
)

var provisionPhaseCommand = &cobra.Command{
	Use:     "provision <CLUSTER_NAME>",
	Short:   "provisions all nodes with the current tools",
	Args:    cobra.ExactArgs(1),
	PreRunE: validateClusterInArgumentExists,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, clusterManager, coordinator := getCommonPhaseDependencies(20, cmd, args)

		phase := phases.NewProvisionNodesPhase(clusterManager)

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
	phaseCommand.AddCommand(provisionPhaseCommand)
}
