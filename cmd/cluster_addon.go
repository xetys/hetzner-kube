package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg/addons"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

// clusterAddonCmd represents the cluster addon command
var clusterAddonCmd = &cobra.Command{
	Use:   "addon",
	Short: "manages addons for kubernetes clusters",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	clusterCmd.AddCommand(clusterAddonCmd)
}

func validateAddonSubCommand(cmd *cobra.Command, args []string) error {
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return nil
	}

	if name == "" {
		return errors.New("flag --name is required")
	}

	idx, cluster := AppConf.Config.FindClusterByName(name)

	if idx == -1 {
		return fmt.Errorf("cluster '%s' not found", name)
	}
	if len(args) != 1 {
		return errors.New("exactly one argument expected")
	}
	addonName := args[0]
	provider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
	addonService := addons.NewClusterAddonService(provider, AppConf.SSHClient)
	if !addonService.AddonExists(addonName) {
		return fmt.Errorf("addon %s not found", addonName)
	}
	return nil
}
