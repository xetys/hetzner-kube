package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/cmd/phases"
	"github.com/xetys/hetzner-kube/pkg"
)

func init() {
	command := declarePhaseCommand("etcd", "setups a etcd cluster", func(cmd *cobra.Command, args []string) {
		provider, clusterManager, coordinator := getCommonPhaseDependencies(6, cmd, args)
		keepData, _ := cmd.Flags().GetBool("keep-data")

		phase := phases.NewEtcdSetupPhase(clusterManager, provider, phases.EtcdSetupPhaseOptions{KeepData: keepData})

		if phase.ShouldRun() {
			err := phase.Run()
			FatalOnError(err)
		}

		for _, node := range provider.GetAllNodes() {
			coordinator.AddEvent(node.Name, pkg.CompletedEvent)
		}

		coordinator.Wait()
	})

	command.Flags().BoolP("keep-data", "k", false, "if set, the old data dir is not deleted")
}
