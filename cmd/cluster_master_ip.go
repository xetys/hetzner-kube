package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// clusterAddWorkerCmd represents the clusterAddWorker command
var clusterMasterIPCmd = &cobra.Command{
	Use:   "master-ip <clustername>",
	Short: "get master node ip",
	Long:  `Returns the IP of the master node. If it's a HA cluster, the IP of the first master will be returned'`,
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if name == "" {
			return errors.New("name is required")
		}

		idx, _ := AppConf.Config.FindClusterByName(name)

		if idx == -1 {
			return fmt.Errorf("cluster '%s' not found", name)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		_, cluster := AppConf.Config.FindClusterByName(name)
		for _, node := range cluster.Nodes {
			if node.IsMaster {
				fmt.Println(node.IPAddress)
				break
			}
		}
	},
}

func init() {
	clusterCmd.AddCommand(clusterMasterIPCmd)
}
