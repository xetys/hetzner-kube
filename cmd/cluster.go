package cmd

import (
	"github.com/spf13/cobra"
)

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "view and manage kubernetes clusters",
	Long: `This command bundles several sub-commands to handle with kubernetes clusters, running on Hetzner Cloud.

Currently it's only supposed to create simple clusters. Upcoming features like separate etcd nodes, multiple masters, upgrades etc.
are hopefully coming soon.'`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	rootCmd.AddCommand(clusterCmd)
}
