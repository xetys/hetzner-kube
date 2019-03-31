package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/cmd/phases"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

func init() {
	declarePhaseCommand("restart", "restart kubelet and docker", func(cmd *cobra.Command, args []string) {
		clusterName := args[0]
		_, cluster := AppConf.Config.FindClusterByName(clusterName)
		provider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
		masterNode, err := provider.GetMasterNode()
		FatalOnError(err)
		err = AppConf.SSHClient.(*clustermanager.SSHCommunicator).CapturePassphrase(masterNode.SSHKeyName)
		FatalOnError(err)

		phase := phases.NewKubeRestartPhase(provider, AppConf.SSHClient)

		err = phase.Run()
		FatalOnError(err)
	})
}
