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
	"strings"
)

// clusterAddWorkerCmd represents the clusterAddWorker command
var clusterAddExternalWorkerCmd = &cobra.Command{
	Use:   "add-external-worker",
	Short: "adds an existing server to the cluster",
	Long: `This lets you add an external server to your cluster.

An external server must meet the following requirements:
	- ubuntu 16.04
	- a unique hostname, that doesn't collide with an existing node name
	- accessible with the same SSH key as used for the cluster`,
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
			return errors.New(fmt.Sprintf("cluster '%s' not found", name))
		}

		ipAddress, _ := cmd.Flags().GetString("ip")

		if ipAddress == "" {
			return errors.New("IP address cannot be empty")
		}

		sshPort, _ := cmd.Flags().GetString("port")

		if sshPort == "" {
			sshPort = "22"
		}

		if len(cluster.Nodes) == 0 {
			return errors.New("your cluster has no nodes, no idea how this was possible")
		}

		externalNode := Node{
			IPAddress:  ipAddress,
			SSHKeyName: cluster.Nodes[0].SSHKeyName,
			SSHPort:    sshPort,
		}

		sshKeyName := cluster.Nodes[0].SSHKeyName
		err = capturePassphrase(sshKeyName)
		if err != nil {
			return err
		}

		// check the host name
		hostname, err := runCmd(externalNode, "hostname -s")
		hostname = strings.TrimSpace(hostname)
		// this also implies the check that SSH is working
		if err != nil {
			return err
		}

		for _, node := range cluster.Nodes {
			if node.Name == hostname {
				return errors.New(fmt.Sprintf("there is already a node with the name '%s'", hostname))
			}
		}

		// check ubuntu 16.04
		issue, err := runCmd(externalNode, "cat /etc/issue | xargs")
		if err != nil {
			return err
		}
		if !strings.Contains(issue, "Ubuntu 16.04") {
			return errors.New("target server has no Ubuntu 16.04 installed")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		ipAddress, _ := cmd.Flags().GetString("ip")
		sshPort, _ := cmd.Flags().GetString("port")
		_, cluster := AppConf.Config.FindClusterByName(name)
		var sshKeyName string

		for _, node := range cluster.Nodes {
			if node.IsMaster {
				sshKeyName = node.SSHKeyName
			}
		}

		if sshPort == "" {
			sshPort = "22"
		}

		if sshKeyName == "" {
			log.Fatal("master not found")
		}

		err := capturePassphrase(sshKeyName)

		if err != nil {
			log.Fatal(err)
		}

		externalNode := Node{
			IPAddress:  ipAddress,
			SSHKeyName: sshKeyName,
			SSHPort:    sshPort,
		}

		hostname, err := runCmd(externalNode, "hostname -s")
		hostname = strings.TrimSpace(hostname)
		FatalOnError(err)
		externalNode.Name = hostname

		// render internal IP address
		nextNode := 21
		for _, node := range cluster.Nodes {
			if !node.IsMaster && !node.IsEtcd {
				nextNode++
			}
		}
		externalNode.PrivateIPAddress = fmt.Sprintf("10.0.1.%d", nextNode)
		cluster.coordinator = pkg.NewProgressCoordinator()

		nodes := []Node{externalNode}

		FatalOnError(err)

		cluster.RenderProgressBars(nodes)
		err = cluster.ProvisionNodes(nodes)
		FatalOnError(err)

		saveCluster(cluster)

		err = cluster.RemoveHCloudManagerKubeletOption(nodes)
		FatalOnError(err)

		cluster.Nodes = append(cluster.Nodes, externalNode)

		// re-generate network encryption
		err = cluster.SetupEncryptedNetwork()
		FatalOnError(err)
		saveCluster(cluster)

		if cluster.HaEnabled {
			err = cluster.DeployLoadBalancer(nodes)
			FatalOnError(err)
		}

		cluster.InstallWorkers(nodes)

		cluster.coordinator.Wait()
		saveCluster(cluster)
		log.Printf("external worker %s with IP %s added to the cluster", externalNode.Name, externalNode.IPAddress)
		log.Println()
	},
}

func init() {
	clusterCmd.AddCommand(clusterAddExternalWorkerCmd)
	clusterAddExternalWorkerCmd.Flags().StringP("name", "n", "", "Name of the cluster to add the workers to")
	clusterAddExternalWorkerCmd.Flags().StringP("ip", "i", "", "The IP address of the external node")
	clusterAddExternalWorkerCmd.Flags().StringP("port", "p", "", "The ssh port of the external node (default 22)")

}
