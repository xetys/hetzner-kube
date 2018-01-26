// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"errors"
	"log"
)

// clusterDeleteCmd represents the clusterDelete command
var clusterDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "removes a cluster and deletes the associated nodes",
	PreRunE: func(cmd *cobra.Command, args []string) error {

		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return nil
		}

		if name == "" {
			return errors.New("flag --name is required")
		}

		idx, _ := AppConf.Config.FindClusterByName(name)

		if idx == -1 {
			return errors.New(fmt.Sprintf("cluster '%s' not found", name))
		}

		if err != nil {
			return err
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		name, _ := cmd.Flags().GetString("name")
		_, cluster := AppConf.Config.FindClusterByName(name)
		// first kill all nodes

		for _, node := range cluster.Nodes {
			server, _, err := AppConf.Client.Server.Get(AppConf.Context, node.Name)

			if err != nil {
				log.Fatal(err)
			}

			if server != nil {
				_, err = AppConf.Client.Server.Delete(AppConf.Context, server)

				if err != nil {
					log.Fatal(err)
				}

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
	clusterDeleteCmd.Flags().StringP("name", "n", "", "Name of the cluster to delete")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clusterDeleteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clusterDeleteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
