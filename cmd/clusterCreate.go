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
	"github.com/xetys/hetzner-kube/pkg"
	"log"
	"time"
	"os"
)

// clusterCreateCmd represents the clusterCreate command
var clusterCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "creates a cluster",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PreRunE: validateClusterCreateFlags,
	Run: func(cmd *cobra.Command, args []string) {

		nodeCount, _ := cmd.Flags().GetInt("nodes")
		workerCount := nodeCount - 1

		clusterName := randomName()
		if name, _ := cmd.Flags().GetString("name"); name != "" {
			clusterName = name
		}

		sshKeyName, _ := cmd.Flags().GetString("ssh-key")
		masterServerType, _ := cmd.Flags().GetString("master-server-type")
		workerServerType, _ := cmd.Flags().GetString("worker-server-type")
		datacenters, _ := cmd.Flags().GetStringSlice("datacenters")

		err := capturePassphrase(sshKeyName)

		if err != nil {
			log.Fatal(err)
		}

		cluster := Cluster{Name: clusterName, wait: false}

		if cloudInit, _ := cmd.Flags().GetString("cloud-init"); cloudInit != "" {
			cluster.CloudInitFile = cloudInit
		}

		if err := cluster.CreateMasterNodes(sshKeyName, masterServerType, datacenters, 1); err != nil {
			log.Println(err)
		}

		var nodes []Node
		if workerCount > 0 {
			var err error
			nodes, err = cluster.CreateWorkerNodes(sshKeyName, workerServerType, datacenters, workerCount, 0)
			FatalOnError(err)
		}

		if cluster.wait {
			log.Println("sleep for 30s...")
			time.Sleep(30 * time.Second)
		}
		cluster.coordinator = pkg.NewProgressCoordinator()
		cluster.RenderProgressBars(cluster.Nodes)

		// provision nodes
		tries := 0
		for err := cluster.ProvisionNodes(nodes); err != nil; {
			if tries < 3 {
				fmt.Print(err)
				tries++
			} else {
				log.Fatal(err)
			}
		}

		// install master
		if err := cluster.InstallMaster(); err != nil {
			log.Fatal(err)
		}

		saveCluster(&cluster)

		// install worker
		if err := cluster.InstallWorkers(cluster.Nodes); err != nil {
			log.Fatal(err)
		}

		cluster.coordinator.Wait()
		log.Println("Cluster successfully created!")

		saveCluster(&cluster)
	},
}

func saveCluster(cluster *Cluster) {
	AppConf.Config.AddCluster(*cluster)
	AppConf.Config.WriteCurrentConfig()
}

func (cluster *Cluster) RenderProgressBars(nodes []Node) {
	for _, node := range nodes {
		steps := 0
		if node.IsMaster {
			// the InstallMaster routine has 9 events
			steps += 9

			// and one more, it's got tainted
			if len(cluster.Nodes) == 1 {
				steps += 1
			}
		} else {
			steps = 4
		}

		cluster.coordinator.StartProgress(node.Name, steps)
	}
}

func validateClusterCreateFlags(cmd *cobra.Command, args []string) error {

	var (
		ssh_key, master_server_type, worker_server_type, cloud_init string
	)

	if ssh_key, _ = cmd.Flags().GetString("ssh-key"); ssh_key == "" {
		return errors.New("flag --ssh-key is required")
	}

	if master_server_type, _ = cmd.Flags().GetString("master-server-type"); master_server_type == "" {
		return errors.New("flag --master_server_type is required")
	}

	if worker_server_type, _ = cmd.Flags().GetString("worker-server-type"); worker_server_type == "" {
		return errors.New("flag --worker_server_type is required")
	}

	if cloud_init, _ = cmd.Flags().GetString("cloud-init"); cloud_init != "" {
		if _, err := os.Stat(cloud_init); os.IsNotExist(err) {
			return errors.New("cloud-init file not found")
		}
	}

	if index, _ := AppConf.Config.FindSSHKeyByName(ssh_key); index == -1 {
		return errors.New(fmt.Sprintf("SSH key '%s' not found", ssh_key))
	}

	return nil
}

func init() {
	clusterCmd.AddCommand(clusterCreateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clusterCreateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	clusterCreateCmd.Flags().String("name", "", "Name of the cluster")
	clusterCreateCmd.Flags().String("ssh-key", "", "Name of the SSH key used for provisioning")
	clusterCreateCmd.Flags().String("master-server-type", "cx11", "Server type used of masters")
	clusterCreateCmd.Flags().String("worker-server-type", "cx11", "Server type used of workers")
	clusterCreateCmd.Flags().Bool("self-hosted", false, "If true, the kubernetes control plane will be hosted on itself")
	clusterCreateCmd.Flags().IntP("nodes", "n", 2, "Number of nodes for the cluster")
	clusterCreateCmd.Flags().StringP("cloud-init", "", "", "Cloud-init file for server preconfiguration")
	clusterCreateCmd.Flags().StringSlice("datacenters", []string{"nbg1-dc3", "fsn1-dc8"}, "Can be used to filter datacenters by their name")
}
