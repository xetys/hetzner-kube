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
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

// clusterCreateCmd represents the clusterCreate command
var clusterCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "creates a cluster",
	Long: `This command lets you create kubernetes clusters with different level of high-availability.

The most simple command is: hetzner-kube cluster create -k YOUR-SSH-KEY-NAME
This will create a 2 node cluster with a random name.

You can specify a name using -n or --name.

= High-Availability =
This tool supports these levels of kubernetes HA:
	level 0: N/A # you cannot create a single-node cluster (currently)
	level 1: hetzner-kube cluster create -k XX -w 3 # distinct masters and 3 workers
	level 2: N/A # you cannot create a non-HA cluster with a separate etcd cluster (currently)
	level 3: hetzner-kube cluster create -k XX -m 3 -w 3 --ha-enabled # deploys a 3 node etcd cluster and a 3-master-node cluster with 3 workers
	level 4: hetzner-kube cluster create -k XX -e 3 -m 2 -w 3 --ha-enabled --isolated-etcd # etcd outside the k8s cluster


	`,
	PreRunE: validateClusterCreateFlags,
	Run:     RunClusterCreate,
}

// RunClusterCreate executes the cluster creation
func RunClusterCreate(cmd *cobra.Command, args []string) {
	workerCount, _ := cmd.Flags().GetInt("worker-count")
	masterCount, _ := cmd.Flags().GetInt("master-count")
	etcdCount := 0
	haEnabled, _ := cmd.Flags().GetBool("ha-enabled")
	if !haEnabled {
		masterCount = 1
	}
	isolatedEtcd, _ := cmd.Flags().GetBool("isolated-etcd")
	if isolatedEtcd {
		etcdCount, _ = cmd.Flags().GetInt("etcd-count")
	}

	clusterName := randomName()
	if name, _ := cmd.Flags().GetString("name"); name != "" {
		clusterName = name
	}

	log.Printf("Creating new cluster\n\nNAME:%s\nMASTERS: %d\nWORKERS: %d\nETCD NODES: %d\nHA: %t\nISOLATED ETCD: %t", clusterName, masterCount, workerCount, etcdCount, haEnabled, isolatedEtcd)

	sshKeyName, _ := cmd.Flags().GetString("ssh-key")
	masterServerType, _ := cmd.Flags().GetString("master-server-type")
	workerServerType, _ := cmd.Flags().GetString("worker-server-type")
	datacenters, _ := cmd.Flags().GetStringSlice("datacenters")
	nodeCidr, _ := cmd.Flags().GetString("node-cidr")
	cloudInit, _ := cmd.Flags().GetString("cloud-init")
	k8sVersion, _ := cmd.Flags().GetString("k8s-version")

	hetznerProvider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, clustermanager.Cluster{
		Name:          clusterName,
		NodeCIDR:      nodeCidr,
		CloudInitFile: cloudInit,
	}, AppConf.CurrentContext.Token)

	sshClient := clustermanager.NewSSHCommunicator(AppConf.Config.SSHKeys)
	err := sshClient.(*clustermanager.SSHCommunicator).CapturePassphrase(sshKeyName)
	FatalOnError(err)

	if haEnabled && isolatedEtcd {
		if _, err := hetznerProvider.CreateEtcdNodes(sshKeyName, masterServerType, datacenters, etcdCount); err != nil {
			log.Println(err)
		}
	}

	if _, err := hetznerProvider.CreateMasterNodes(sshKeyName, masterServerType, datacenters, masterCount, !isolatedEtcd); err != nil {
		log.Println(err)
	}

	if workerCount > 0 {
		var err error
		_, err = hetznerProvider.CreateWorkerNodes(sshKeyName, workerServerType, datacenters, workerCount, 0)
		FatalOnError(err)
	}

	if hetznerProvider.MustWait() {
		log.Println("sleep for 10s...")
		time.Sleep(10 * time.Second)
	}

	coordinator := pkg.NewProgressCoordinator()

	clusterManager := clustermanager.NewClusterManager(hetznerProvider, sshClient, coordinator, clusterName, haEnabled, isolatedEtcd, cloudInit, false, k8sVersion)
	cluster := clusterManager.Cluster()
	saveCluster(&cluster)
	renderProgressBars(&cluster, coordinator)

	// provision nodes
	tries := 0
	for err := clusterManager.ProvisionNodes(cluster.Nodes); err != nil; {
		if tries < 3 {
			fmt.Print(err)
			tries++
		} else {
			log.Fatal(err)
		}
	}

	// setup encrypted network
	err = clusterManager.SetupEncryptedNetwork()
	FatalOnError(err)
	cluster = clusterManager.Cluster()
	saveCluster(&cluster)

	if haEnabled {
		var etcdNodes []clustermanager.Node

		if isolatedEtcd {
			etcdNodes = hetznerProvider.GetEtcdNodes()
		} else {
			etcdNodes = hetznerProvider.GetMasterNodes()
		}

		err = clusterManager.InstallEtcdNodes(etcdNodes)
		FatalOnError(err)

		saveCluster(&cluster)
	}

	// install masters
	if err := clusterManager.InstallMasters(); err != nil {
		log.Fatal(err)
	}

	// ha plane
	if haEnabled {
		err = clusterManager.SetupHA()
		FatalOnError(err)
	}

	// install worker
	if err := clusterManager.InstallWorkers(cluster.Nodes); err != nil {
		log.Fatal(err)
	}

	coordinator.Wait()
	log.Println("Cluster successfully created!")
}

func saveCluster(cluster *clustermanager.Cluster) {
	AppConf.Config.AddCluster(*cluster)
	AppConf.Config.WriteCurrentConfig()
}

func renderProgressBars(cluster *clustermanager.Cluster, coordinator *pkg.UIProgressCoordinator) {
	nodes := cluster.Nodes
	provisionSteps := 5
	netWorkSetupSteps := 2
	etcdSteps := 4
	masterInstallSteps := 2
	numMaster := 0
	for _, node := range nodes {
		steps := provisionSteps + netWorkSetupSteps
		if node.IsEtcd {
			steps += etcdSteps
		}

		if node.IsMaster {
			numMaster++
			steps += masterInstallSteps
			steps += computeMasterSteps(numMaster, cluster)
		}

		if !node.IsEtcd && !node.IsMaster {
			steps = computeWorkerSteps(steps, cluster)
		}

		coordinator.StartProgress(node.Name, steps+6)
	}
}

func computeWorkerSteps(steps int, cluster *clustermanager.Cluster) int {
	workerHaSteps := 1
	nodeInstallSteps := 1
	steps += nodeInstallSteps
	if cluster.HaEnabled {
		steps += workerHaSteps
	}
	return steps
}

func computeMasterSteps(numMaster int, cluster *clustermanager.Cluster) int {
	masterFirstSteps := 4
	masterHaNonFirstSteps := 1
	masterHaSteps := 4
	steps := 0
	// the InstallMasters routine has 9 events
	if numMaster == 1 {
		steps += masterFirstSteps
	}
	if numMaster > 1 && cluster.HaEnabled {
		steps += masterHaNonFirstSteps
	}
	if cluster.HaEnabled {
		steps += masterHaSteps
	}
	// and one more, it's got tainted
	if len(cluster.Nodes) == 1 {
		steps++
	}
	return steps
}

func validateClusterCreateFlags(cmd *cobra.Command, args []string) error {

	var (
		sshKey, masterServerType, workerServerType, cloudInit string
	)

	if sshKey, _ = cmd.Flags().GetString("ssh-key"); sshKey == "" {
		return errors.New("flag --ssh-key is required")
	}

	if masterServerType, _ = cmd.Flags().GetString("master-server-type"); masterServerType == "" {
		return errors.New("flag --master_server_type is required")
	}

	if workerServerType, _ = cmd.Flags().GetString("worker-server-type"); workerServerType == "" {
		return errors.New("flag --worker_server_type is required")
	}

	if nodeCidr, _ := cmd.Flags().GetString("node-cidr"); nodeCidr != "10.0.1.0/24" {
		_, _, err := net.ParseCIDR(nodeCidr)

		if err != nil {
			return fmt.Errorf("could not parse cidr: %v", err)
		}
	}

	if cloudInit, _ = cmd.Flags().GetString("cloud-init"); cloudInit != "" {
		if _, err := os.Stat(cloudInit); os.IsNotExist(err) {
			return errors.New("cloud-init file not found")
		}
	}

	if _, err := AppConf.Config.FindSSHKeyByName(sshKey); err != nil {
		return fmt.Errorf("SSH key '%s' not found", sshKey)
	}

	haEnabled, _ := cmd.Flags().GetBool("ha-enabled")
	isolatedEtcd, _ := cmd.Flags().GetBool("isolated-etcd")

	if worker, _ := cmd.Flags().GetInt("worker-count"); worker < 1 {
		return fmt.Errorf("at least 1 worker node is needed. %d was provided", worker)
	}

	if haEnabled {
		if isolatedEtcd {
			if master, _ := cmd.Flags().GetInt("master-count"); master < 2 {
				return fmt.Errorf("at least 2 master node are needed. %d was provided", master)
			}

			if etcds, _ := cmd.Flags().GetInt("etcd-count"); etcds%2 == 0 || etcds < 3 {
				return fmt.Errorf("the number of etcds should be odd and at least 3. %d was provided", etcds)
			}
		} else {
			if master, _ := cmd.Flags().GetInt("master-count"); master < 3 {
				return fmt.Errorf("at least 3 master node are needed when etcd is installed on them. %d was provided", master)
			}

			if etcds, _ := cmd.Flags().GetInt("etcd-count"); etcds != 3 {
				return errors.New("you cannot use etcd count without --isolated-etcd")
			}
		}
	}

	return nil
}

func init() {
	clusterCmd.AddCommand(clusterCreateCmd)

	clusterCreateCmd.Flags().StringP("name", "n", "", "Name of the cluster")
	clusterCreateCmd.Flags().StringP("ssh-key", "k", "", "Name of the SSH key used for provisioning")
	clusterCreateCmd.Flags().String("master-server-type", "cx11", "Server type used of masters")
	clusterCreateCmd.Flags().String("worker-server-type", "cx11", "Server type used of workers")
	clusterCreateCmd.Flags().String("k8s-version", "1.9.6-00", "The version of the k8s debian packages that will be used during provisioning")
	clusterCreateCmd.Flags().Bool("ha-enabled", false, "Install high-available control plane")
	clusterCreateCmd.Flags().Bool("isolated-etcd", false, "Isolates etcd cluster from master nodes")
	clusterCreateCmd.Flags().IntP("master-count", "m", 3, "Number of master nodes, works only if -ha-enabled is passed")
	clusterCreateCmd.Flags().IntP("etcd-count", "e", 3, "Number of etcd nodes, works only if --ha-enabled and --isolated-etcd are passed")
	clusterCreateCmd.Flags().Bool("self-hosted", false, "If true, the kubernetes control plane will be hosted on itself")
	clusterCreateCmd.Flags().IntP("worker-count", "w", 1, "Number of worker nodes for the cluster")
	clusterCreateCmd.Flags().StringP("cloud-init", "", "", "Cloud-init file for server preconfiguration")
	clusterCreateCmd.Flags().StringP("node-cidr", "", "10.0.1.0/24", "the CIDR for the nodes wireguard IPs")
	clusterCreateCmd.Flags().StringSlice("datacenters", []string{"nbg1-dc3", "fsn1-dc8", "hel1-dc2"}, "Can be used to filter datacenters by their name")
}
