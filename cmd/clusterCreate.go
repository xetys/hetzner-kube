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

		workerCount, _ := cmd.Flags().GetInt("worker-count")
		masterCount, _ := cmd.Flags().GetInt("master-count")
		etcdCount, _ := cmd.Flags().GetInt("etcd-count")
		haEnabled, _ := cmd.Flags().GetBool("ha-enabled")
		isolatedEtcd, _ := cmd.Flags().GetBool("iso-etcd")

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

		cluster := Cluster{Name: clusterName, wait: false, HaEnabled: haEnabled, IsolatedEtcd: isolatedEtcd}

		if cloudInit, _ := cmd.Flags().GetString("cloud-init"); cloudInit != "" {
			cluster.CloudInitFile = cloudInit
		}

		if haEnabled && isolatedEtcd {
			if err := cluster.CreateEtcdNodes(sshKeyName, masterServerType, datacenters, etcdCount); err != nil {
				log.Println(err)
			}
		}

		if err := cluster.CreateMasterNodes(sshKeyName, masterServerType, datacenters, masterCount); err != nil {
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

		// setup encrypted network
		err = cluster.SetupEncryptedNetwork()
		FatalOnError(err)
		saveCluster(&cluster)

		if haEnabled {
			var etcdNodes []Node

			if isolatedEtcd {
				etcdNodes = cluster.GetEtcdNodes()
			} else {
				etcdNodes = cluster.GetMasterNodes()
			}

			err = cluster.InstallEtcdNodes(etcdNodes)
			FatalOnError(err)

			saveCluster(&cluster)
		}

		// install masters
		if err := cluster.InstallMasters(); err != nil {
			log.Fatal(err)
		}

		saveCluster(&cluster)

		// ha plane
		if haEnabled {
			err = cluster.SetupHA()
			FatalOnError(err)
			time.Sleep(30 * time.Second)
		}

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
	provisionSteps := 2
	netWorkSetupSteps := 2
	etcdSteps := 4
	masterInstallSteps := 2
	masterNonHaSteps := 4
	masterHaNonFirstSteps := 1
	masterHaSteps := 4
	workerHaSteps := 1
	nodeInstallSteps := 1
	for idx, node := range nodes {
		steps := provisionSteps + netWorkSetupSteps
		if node.IsEtcd {
			steps += etcdSteps
		}
		if node.IsMaster {
			// the InstallMasters routine has 9 events
			steps += masterInstallSteps
			if idx == 0 {
				steps += masterNonHaSteps
			}

			if idx > 0 && cluster.HaEnabled {
				steps += masterHaNonFirstSteps
			}

			if cluster.HaEnabled {
				steps += masterHaSteps
			}

			// and one more, it's got tainted
			if len(cluster.Nodes) == 1 {
				steps += 1
			}
		} else {
			steps += nodeInstallSteps

			if cluster.HaEnabled {
				steps += workerHaSteps
			}
		}

		cluster.coordinator.StartProgress(node.Name, 100)
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

	haEnabled, _ := cmd.Flags().GetBool("ha-enabled")
	isolatedEtcd, _ := cmd.Flags().GetBool("iso-etcd")

	if worker, _ := cmd.Flags().GetInt("worker-count"); worker < 1 {
		return errors.New(fmt.Sprintf("at least 1 worker node is needed. %d was provided", worker))
	}

	if haEnabled {
		if isolatedEtcd {
			if master, _ := cmd.Flags().GetInt("master-count"); master < 2 {
				return errors.New(fmt.Sprintf("at least 2 master node are needed. %d was provided", master))
			}

			if etcds, _ := cmd.Flags().GetInt("etcd-count"); etcds%2 == 0 || etcds < 3 {
				return errors.New(fmt.Sprintf("the number of etcds should be odd and at least 3. %d was provided", etcds))
			}
		} else {
			if master, _ := cmd.Flags().GetInt("master-count"); master < 3 {
				return errors.New(fmt.Sprintf("at least 3 master node are needed when etcd is installed on them. %d was provided", master))
			}

			if etcds, _ := cmd.Flags().GetInt("etcd-count"); etcds != 3 {
				return errors.New("you cannot use etcd count without --iso-etcd")
			}
		}
	}

	return nil
}

func init() {
	clusterCmd.AddCommand(clusterCreateCmd)

	clusterCreateCmd.Flags().String("name", "", "Name of the cluster")
	clusterCreateCmd.Flags().StringP("ssh-key", "k", "", "Name of the SSH key used for provisioning")
	clusterCreateCmd.Flags().String("master-server-type", "cx11", "Server type used of masters")
	clusterCreateCmd.Flags().String("worker-server-type", "cx11", "Server type used of workers")
	clusterCreateCmd.Flags().Bool("ha-enabled", false, "Install high-available control plane")
	clusterCreateCmd.Flags().Bool("iso-etcd", false, "Isolates etcd cluster from master nodes")
	clusterCreateCmd.Flags().Int("master-count", 3, "Number of master nodes, works only if -ha-enabled is passed")
	clusterCreateCmd.Flags().Int("etcd-count", 3, "Number of etcd nodes, works only if --ha-enabled and --iso-etcd are passed")
	clusterCreateCmd.Flags().Bool("self-hosted", false, "If true, the kubernetes control plane will be hosted on itself")
	clusterCreateCmd.Flags().IntP("worker-count", "w", 1, "Number of worker nodes for the cluster")
	clusterCreateCmd.Flags().StringP("cloud-init", "", "", "Cloud-init file for server preconfiguration")
	clusterCreateCmd.Flags().StringSlice("datacenters", []string{"nbg1-dc3", "fsn1-dc8"}, "Can be used to filter datacenters by their name")
}
