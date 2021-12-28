package cmd

import (
	"errors"
	"fmt"

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
	clusterCmd.PersistentFlags().StringP("k8s-version", "v", "v1.22.4+rke2r2", "the version of Kubernetes to be installed")
	rootCmd.AddCommand(clusterCmd)
}

// validateClusterInArgumentExists checks if a cluster name is present in arguments and the cluster can be found
func validateClusterInArgumentExists(cmd *cobra.Command, args []string) error {

	name := args[0]

	if name == "" {
		return errors.New("argument NAME is required")
	}

	idx, _ := AppConf.Config.FindClusterByName(name)

	if idx == -1 {
		return fmt.Errorf("cluster '%s' not found", name)
	}
	return nil
}
