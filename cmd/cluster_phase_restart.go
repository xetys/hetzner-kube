package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
	phases "github.com/xetys/hetzner-kube/pkg/phases"
)

var kubeRestartPhaseCommand = &cobra.Command{
	Use:     "restart <CLUSTER_NAME>",
	Short:   "restart kubelet and docker",
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

		phase := phases.NewKubeRestartPhase(provider, AppConf.SSHClient)

		return phase.Run()
	},
}

func init() {
	phaseCommand.AddCommand(kubeRestartPhaseCommand)
}
