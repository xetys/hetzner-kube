package cmd

import (
	"errors"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

// clusterRemoveWorkerCmd represents the command for removing workers
var clusterRemoveWorkerCmd = &cobra.Command{
	Use:   "remove-worker",
	Short: "remove an worker from node list",
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

		workerName, _ := cmd.Flags().GetString("worker")

		if workerName == "" {
			return errors.New("worker name cannot be empty")
		}

		for _, node := range cluster.Nodes {
			if node.Name == workerName {
				return nil
			}
		}

		return errors.New("node not found")
	},
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		workerName, _ := cmd.Flags().GetString("worker")
		_, cluster := AppConf.Config.FindClusterByName(name)
		masterVisited := false
		var masterNode clustermanager.Node

		for idx, node := range cluster.Nodes {
			if node.IsMaster && !masterVisited {
				masterNode = node
				masterVisited = true
			}

			if node.Name == workerName {
				// delete actual server
				server, _, err := AppConf.Client.Server.Get(AppConf.Context, node.Name)

				FatalOnError(err)

				if server != nil {
					_, err = AppConf.Client.Server.Delete(AppConf.Context, server)

					FatalOnError(err)

					log.Printf("server '%s' deleted", node.Name)
				} else {
					log.Printf("server '%s' was already deleted", node.Name)
				}

				// remove from k8s
				_, err = AppConf.SSHClient.RunCmd(masterNode, fmt.Sprintf("kubectl delete node %s", node.Name))

				if err != nil {
					log.Printf("deletion failed %s", err)
				}
				cluster.Nodes = append(cluster.Nodes[:idx], cluster.Nodes[idx+1:]...)
				saveCluster(cluster)
			}
		}

		log.Println("node deleted successfully")
	},
}

func init() {
	clusterCmd.AddCommand(clusterRemoveWorkerCmd)

	clusterRemoveWorkerCmd.Flags().StringP("name", "n", "", "Name of the cluster where to remove the worker")
	clusterRemoveWorkerCmd.Flags().StringP("worker", "w", "", "The name of the worker to remove")
}
