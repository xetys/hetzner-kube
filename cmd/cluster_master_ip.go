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
