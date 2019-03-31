package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

var phaseCommand = &cobra.Command{
	Use:   "phase",
	Short: "performs single phases from cluster creation",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func declarePhaseCommand(name string, short string, run func(cmd *cobra.Command, args []string)) *cobra.Command {
	declaredCommand := &cobra.Command{
		Use:     name,
		Short:   short,
		PreRunE: validateClusterInArgumentExists,
		Run:     run,
	}

	phaseCommand.AddCommand(declaredCommand)

	return declaredCommand
}

func getCommonPhaseDependencies(steps int, cmd *cobra.Command, args []string) (clustermanager.ClusterProvider, *clustermanager.Manager, *pkg.UIProgressCoordinator) {
	clusterName := args[0]

	_, cluster := AppConf.Config.FindClusterByName(clusterName)
	provider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
	masterNode, err := provider.GetMasterNode()
	FatalOnError(err)
	err = AppConf.SSHClient.(*clustermanager.SSHCommunicator).CapturePassphrase(masterNode.SSHKeyName)
	FatalOnError(err)
	coordinator := pkg.NewProgressCoordinator()

	for _, node := range provider.GetAllNodes() {
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

	return provider, clusterManager, coordinator
}

func init() {
	clusterCmd.AddCommand(phaseCommand)
}
