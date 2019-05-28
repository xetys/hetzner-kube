package cmd

import (
	"errors"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// clusterDeleteCmd represents the clusterDelete command
var clusterDeleteCmd = &cobra.Command{
	Use:   "delete <CLUSTER_NAME>",
	Short: "removes a cluster and deletes the associated nodes",
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
		// first kill all nodes

		for _, node := range cluster.Nodes {
			server, _, err := AppConf.Client.Server.Get(AppConf.Context, node.Name)

			FatalOnError(err)

			if server != nil {
				_, err = AppConf.Client.Server.Delete(AppConf.Context, server)

				FatalOnError(err)

				log.Printf("server '%s' deleted", node.Name)
			} else {
				log.Printf("server '%s' was already deleted", node.Name)
			}
		}

		// now remove the cluster from list
		if err := AppConf.Config.DeleteCluster(name); err != nil {
			log.Fatal(err)
		}

		AppConf.Config.WriteCurrentConfig()

		log.Printf("cluster '%s' deleted", name)
	},
}

func init() {
	clusterCmd.AddCommand(clusterDeleteCmd)
}
