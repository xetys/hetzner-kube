package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg/addons"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

// clusterAddonInstallCmd represents the clusterAddonInstall command
var clusterAddonUninstallCmd = &cobra.Command{
	Use:     "uninstall",
	Short:   "removes an addon to a cluster",
	PreRunE: validateAddonSubCommand,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		addonName := args[0]

		_, cluster := AppConf.Config.FindClusterByName(name)

		log.Printf("removing addon %s", addonName)
		provider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
		masterNode, err := provider.GetMasterNode()
		FatalOnError(err)

		err = AppConf.SSHClient.(*clustermanager.SSHCommunicator).CapturePassphrase(masterNode.SSHKeyName)
		FatalOnError(err)

		addonService := addons.NewClusterAddonService(provider, AppConf.SSHClient)
		addon := addonService.GetAddon(addonName)
		addon.Uninstall()

		log.Printf("addon %s successfully removed", addonName)
	},
}

func init() {
	clusterAddonCmd.AddCommand(clusterAddonUninstallCmd)

	clusterAddonUninstallCmd.Flags().StringP("name", "n", "", "Name of the cluster")
}
