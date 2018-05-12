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
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

// clusterAddWorkerCmd represents the clusterAddWorker command
var clusterAddWorkerCmd = &cobra.Command{
	Use:   "add-worker",
	Short: "add worker nodes",
	Long: `Adds n nodes as worker nodes to the cluster.
You can specify the worker server type as in cluster create.`,
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
			return fmt.Errorf("cluster '%s' not found", name)
		}

		var workerServerType string
		if workerServerType, _ = cmd.Flags().GetString("worker-server-type"); workerServerType == "" {
			return errors.New("flag --worker-server-type is required")
		}

		if err != nil {
			return err
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		nodeCount, _ := cmd.Flags().GetInt("nodes")
		name, _ := cmd.Flags().GetString("name")
		_, cluster := AppConf.Config.FindClusterByName(name)
		workerServerType, _ := cmd.Flags().GetString("worker-server-type")
		datacenters, _ := cmd.Flags().GetStringSlice("datacenters")
		var sshKeyName string

		for _, node := range cluster.Nodes {
			if node.IsMaster {
				sshKeyName = node.SSHKeyName
			}
		}

		if sshKeyName == "" {
			log.Fatal("master not found")
		}

		maxNo := 0
		for _, node := range cluster.Nodes {
			if !node.IsMaster {
				nameParts := strings.Split(node.Name, "-")
				no, _ := strconv.Atoi(nameParts[len(nameParts)-1])

				if no > maxNo {
					maxNo = no
				}
			}
		}

		coordinator := pkg.NewProgressCoordinator()
		hetznerProvider, clusterManager := hetzner.ProviderAndManager(AppConf.Context, *cluster, AppConf.Client, AppConf.SSHClient, coordinator, AppConf.CurrentContext.Token)
		err := AppConf.SSHClient.(*clustermanager.SSHCommunicator).CapturePassphrase(sshKeyName)
		if err != nil {
			log.Fatal(err)
		}

		nodes, err := hetznerProvider.CreateWorkerNodes(sshKeyName, workerServerType, datacenters, nodeCount, maxNo)
		FatalOnError(err)
		saveCluster(cluster)

		log.Println("sleep for 30s...")
		time.Sleep(30 * time.Second)

		renderProgressBars(cluster, coordinator)
		err = clusterManager.ProvisionNodes(nodes)
		FatalOnError(err)

		// re-generate network encryption
		err = clusterManager.SetupEncryptedNetwork()
		FatalOnError(err)
		saveCluster(cluster)

		if cluster.HaEnabled {
			err = clusterManager.DeployLoadBalancer(nodes)
			FatalOnError(err)
		}

		clusterManager.InstallWorkers(nodes)

		coordinator.Wait()
		log.Println("workers created successfully")
	},
}

func init() {
	clusterCmd.AddCommand(clusterAddWorkerCmd)
	clusterAddWorkerCmd.Flags().StringP("name", "", "", "Name of the cluster to add the workers to")
	clusterAddWorkerCmd.Flags().String("worker-server-type", "cx11", "Server type used of workers")
	clusterAddWorkerCmd.Flags().IntP("nodes", "n", 2, "Number of nodes for the cluster")
	clusterAddWorkerCmd.Flags().StringSlice("datacenters", []string{"nbg1-dc3", "fsn1-dc8"}, "Can be used to filter datacenters by their name")
}
