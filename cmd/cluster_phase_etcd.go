package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg"
	phases "github.com/xetys/hetzner-kube/pkg/phases"
)

var etcdPhaseCommand = &cobra.Command{
	Use:     "etcd <CLUSTER_NAME>",
	Short:   "setups a etcd cluster",
	Args:    cobra.ExactArgs(1),
	PreRunE: validateClusterInArgumentExists,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, clusterManager, coordinator := getCommonPhaseDependencies(6, cmd, args)
		keepData, _ := cmd.Flags().GetBool("keep-data")

		phase := phases.NewEtcdSetupPhase(clusterManager, provider, phases.EtcdSetupPhaseOptions{KeepData: keepData})

		if phase.ShouldRun() {
			err := phase.Run()
			if err != nil {
				return err
			}
		}

		for _, node := range provider.GetAllNodes() {
			coordinator.AddEvent(node.Name, pkg.CompletedEvent)
		}

		coordinator.Wait()
		return nil
	},
}

func init() {

	etcdPhaseCommand.Flags().BoolP("keep-data", "k", false, "if set, the old data dir is not deleted")
	phaseCommand.AddCommand(etcdPhaseCommand)
}
