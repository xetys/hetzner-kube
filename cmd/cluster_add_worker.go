package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
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

		var cloudInit string
		if cloudInit, _ = cmd.Flags().GetString("cloud-init"); cloudInit != "" {
			if _, err := os.Stat(cloudInit); os.IsNotExist(err) {
				return errors.New("cloud-init file not found")
			}
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
		cloudInit, _ := cmd.Flags().GetString("cloud-init")
		var sshKeyName string

		for _, node := range cluster.Nodes {
			if node.IsMaster {
				sshKeyName = node.SSHKeyName
				break
			}
		}

		if sshKeyName == "" {
			log.Fatal("master not found")
		}

		if cloudInit != "" {
			cluster.CloudInitFile = cloudInit
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

		coordinator := pkg.NewProgressCoordinator(DebugMode)
		hetznerProvider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
		clusterManager := clustermanager.NewClusterManagerFromCluster(*cluster, hetznerProvider, AppConf.SSHClient, coordinator)
		err := AppConf.SSHClient.(*clustermanager.SSHCommunicator).CapturePassphrase(sshKeyName)
		if err != nil {
			log.Fatal(err)
		}

		nodes, err := hetznerProvider.CreateWorkerNodes(sshKeyName, workerServerType, datacenters, nodeCount, maxNo)
		FatalOnError(err)

		existingNodes := cluster.Nodes

		cluster.Nodes = append(cluster.Nodes, nodes...)
		saveCluster(cluster)

		// Is needed to the right wireguard config is created including the new nodes
		clusterManager.AppendNodes(nodes)

		log.Println("sleep for 30s...")
		time.Sleep(30 * time.Second)

		renderProgressBars(cluster, coordinator)
		err = clusterManager.ProvisionNodes(nodes)
		FatalOnError(err)

		// re-generate network encryption
		err = clusterManager.SetupEncryptedNetwork()
		FatalOnError(err)
		saveCluster(cluster)

		// all work on the already existing nodes is completed by now
		for _, node := range existingNodes {
			coordinator.CompleteProgress(node.Name)
		}

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
	clusterAddWorkerCmd.Flags().StringSlice("datacenters", []string{"fsn1-dc8", "nbg1-dc3", "hel1-dc2", "fsn1-dc14"}, "Can be used to filter datacenters by their name")
	clusterAddWorkerCmd.Flags().StringP("cloud-init", "", "", "Cloud-init file for node preconfiguration")
}
