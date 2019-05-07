package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
	phases "github.com/xetys/hetzner-kube/pkg/phases"
)

var setupHAPhaseCommand = &cobra.Command{
	Use:     "setup-ha <CLUSTER_NAME>",
	Short:   "configures the cluster for high availability",
	Args:    cobra.ExactArgs(1),
	PreRunE: validateClusterInArgumentExists,
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName := args[0]

		_, cluster := AppConf.Config.FindClusterByName(clusterName)
		provider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
		masterNode, err := provider.GetMasterNode()
		if err != nil {
			return err
		}
		err = AppConf.SSHClient.(*clustermanager.SSHCommunicator).CapturePassphrase(masterNode.SSHKeyName)
		if err != nil {
			return err
		}
		coordinator := pkg.NewProgressCoordinator()

		for _, node := range provider.GetAllNodes() {
			steps := 2
			if node.Name == masterNode.Name {
				steps += 4
			} else {
				steps += 2
			}
			coordinator.StartProgress(node.Name, steps)
		}

		clusterManager := clustermanager.NewClusterManager(
			provider,
			AppConf.SSHClient,
			coordinator,
			clusterName,
			cluster.HaEnabled,
			cluster.IsolatedEtcd,
			cluster.CloudInitFile,
		)

		phase := phases.NewSetupHighAvailabilityPhase(clusterManager)

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
	phaseCommand.AddCommand(setupHAPhaseCommand)
}
