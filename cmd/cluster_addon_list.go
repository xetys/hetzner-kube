package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg/addons"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

// clusterAddonInstallCmd represents the clusterAddonInstall command
var clusterAddonListCmd = &cobra.Command{
	Use:   "list",
	Short: "list the currently available addons",
	Run: func(cmd *cobra.Command, args []string) {
		tw := new(tabwriter.Writer)
		tw.Init(os.Stdout, 0, 8, 2, '\t', 0)
		fmt.Fprintln(tw, "NAME\tREQUIRES\tDESCRIPTION\tURL")

		cluster := &clustermanager.Cluster{Nodes: []clustermanager.Node{{IsMaster: true}}}
		provider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
		addonService := addons.NewClusterAddonService(provider, AppConf.SSHClient)
		for _, addon := range addonService.Addons() {
			requires := "-"
			if len(addon.Requires()) > 0 {
				requires = strings.Join(addon.Requires(), ", ")
			}

			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t", addon.Name(), requires, addon.Description(), addon.URL())
			fmt.Fprintln(tw)
		}

		tw.Flush()
	},
}

func init() {
	clusterAddonCmd.AddCommand(clusterAddonListCmd)
}
