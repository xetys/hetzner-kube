package cmd

import (
	"errors"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

// clusterAddWorkerCmd represents the clusterAddWorker command
var clusterRemoveExternalWorkerCmd = &cobra.Command{
	Use:   "remove-external-worker",
	Short: "remove an external worker from node list",
	PreRunE: func(cmd *cobra.Command, args []string) error {
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

		ipAddress, _ := cmd.Flags().GetString("ip")

		if ipAddress == "" {
			return errors.New("IP address cannot be empty")
		}

		for _, node := range cluster.Nodes {
			if node.IPAddress == ipAddress {
				return nil
			}
		}

		return errors.New("node not found")
	},
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		ipAddress, _ := cmd.Flags().GetString("ip")
		_, cluster := AppConf.Config.FindClusterByName(name)
		masterVisited := false
		var masterNode clustermanager.Node

		for idx, node := range cluster.Nodes {
			if node.IsMaster && !masterVisited {
				masterNode = node
				masterVisited = true
			}

			if node.IPAddress == ipAddress {
				_, err := AppConf.SSHClient.RunCmd(masterNode, fmt.Sprintf("kubectl delete node %s", node.Name))

				log.Printf("deletion failed %s", err)
				cluster.Nodes = append(cluster.Nodes[:idx], cluster.Nodes[idx+1:]...)
				saveCluster(cluster)
			}
		}

		log.Println("node deleted successfully")
	},
}

func init() {
	clusterCmd.AddCommand(clusterRemoveExternalWorkerCmd)

	clusterRemoveExternalWorkerCmd.Flags().StringP("name", "n", "", "Name of the cluster where to remove the worker")
	clusterRemoveExternalWorkerCmd.Flags().StringP("ip", "i", "", "The IP address of the external node")
}
