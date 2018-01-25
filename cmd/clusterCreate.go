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
	"github.com/spf13/cobra"
	"log"
	"errors"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"strings"
	"time"
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
		clusterName, _ := cmd.Flags().GetString("name")
		sshKeyName, _ := cmd.Flags().GetString("ssh-key")
		masterServerType, _ := cmd.Flags().GetString("master-server-type")
		workerServerType, _ := cmd.Flags().GetString("worker-server-type")
		cluster := Cluster{Name: clusterName}

		if err := cluster.CreateMasterNodes(Node{SSHKeyName: sshKeyName, IsMaster: true, Type: masterServerType}, 1); err != nil {
			log.Println(err)
		}

		if workerCount > 0 {
			if err := cluster.CreateWorkerNodes(Node{SSHKeyName: sshKeyName, IsMaster: false, Type: workerServerType}, workerCount); err != nil {
				log.Fatal(err)
			}
		}

		log.Println("sleep for 10s...")
		time.Sleep(10 * time.Second)

		// provision nodes
		if err := cluster.ProvisionNodes(); err != nil {
			log.Fatal(err)
		}

		// install master
		if err := cluster.InstallMaster(); err != nil {
			log.Fatal(err)
		}

		// install worker
		if err := cluster.InstallWorkers(); err != nil {
			log.Fatal(err)
		}

		log.Println("Cluster successfully created!")

		AppConf.Config.AddCluster(cluster)
		AppConf.Config.WriteCurrentConfig()
	},
}

func (cluster *Cluster) CreateNodes(suffix string, template Node, count int) error {
	sshKey, _, err := AppConf.Client.SSHKey.Get(AppConf.Context, template.SSHKeyName)

	if err != nil {
		return err
	}

	serverNameTemplate := fmt.Sprintf("%s-%s-@idx", cluster.Name, suffix)
	serverOptsTemplate := hcloud.ServerCreateOpts{
		Name: serverNameTemplate,
		ServerType: &hcloud.ServerType{
			Name: template.Type,
		},
		Image: &hcloud.Image{
			Name: "ubuntu-16.04",
		},
	}

	serverOptsTemplate.SSHKeys = append(serverOptsTemplate.SSHKeys, sshKey)

	for i := 1; i <= count; i++ {
		var serverOpts hcloud.ServerCreateOpts
		serverOpts = serverOptsTemplate
		serverOpts.Name = strings.Replace(serverNameTemplate, "@idx", fmt.Sprintf("%.02d", i), 1)

		// create
		server, err := runCreateServer(&serverOpts)

		if err != nil {
			return err
		}

		ipAddress := server.Server.PublicNet.IPv4.IP.String()
		log.Printf("Created node '%s' with IP %s", server.Server.Name, ipAddress)
		cluster.Nodes = append(cluster.Nodes, Node{
			Name:       serverOpts.Name,
			Type:       serverOpts.ServerType.Name,
			IsMaster:   template.IsMaster,
			IPAddress:  ipAddress,
			SSHKeyName: template.SSHKeyName,
		})
	}

	return nil
}

func runCreateServer(opts *hcloud.ServerCreateOpts) (*hcloud.ServerCreateResult, error) {

	log.Printf("creating server '%s'...", opts.Name)
	result, _, err := AppConf.Client.Server.Create(AppConf.Context, *opts)
	if err != nil {
		if err.(hcloud.Error).Code == "uniqueness_error" {
			server, _, err := AppConf.Client.Server.Get(AppConf.Context, opts.Name)

			if err != nil {
				return nil, err
			}

			log.Printf("loading server '%s'...", opts.Name)
			return &hcloud.ServerCreateResult{Server: server}, nil
		}

		return nil, err
	}

	if err := AppConf.ActionProgress(AppConf.Context, result.Action); err != nil {
		return nil, err
	}

	return &result, nil
}

func (cluster *Cluster) CreateMasterNodes(template Node, count int) error {
	log.Println("creating master nodes...")
	return cluster.CreateNodes("master", template, count)
}

func (cluster *Cluster) CreateWorkerNodes(template Node, count int) error {
	return cluster.CreateNodes("worker", template, count)
}

func (cluster *Cluster) ProvisionNodes() error {
	for _, node := range cluster.Nodes {
		log.Printf("installing docker.io and kubeadm on node '%s'...", node.Name)
		_, err := runCmd(node, "wget -cO- https://gist.githubusercontent.com/xetys/0ecfa01790debb2345c0883418dcc7c4/raw/403b6cdea6b78bc5b7209acfa3dfa810dd5f89ba/ubuntu16-kubeadm | bash -")

		if err != nil {
			return err
		}
	}

	return nil
}
func (cluster *Cluster) InstallMaster() error {
	commands := []string{
		"swapoff -a",
		"kubeadm init --pod-network-cidr=192.168.0.0/16",
		"mkdir -p $HOME/.kube",
		"cp -i /etc/kubernetes/admin.conf $HOME/.kube/config",
		"chown $(id -u):$(id -g) $HOME/.kube/config",
		"kubectl apply -f https://docs.projectcalico.org/v2.6/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml",
	}
	for _, node := range cluster.Nodes {
		if node.IsMaster {
			if len(cluster.Nodes) == 1 {
				commands = append(commands, "kubectl taint nodes --all node-role.kubernetes.io/master-")
			}

			for _, command := range commands {
				_, err := runCmd(node, command)
				if err != nil {
					return err
				}
			}

			break
		}
	}

	return nil
}

func (cluster *Cluster) InstallWorkers() error {
	var joinCommand string
	// find master
	for _, node := range cluster.Nodes {
		if node.IsMaster {
			output, err := runCmd(node, "kubeadm token create --print-join-command")
			if err != nil {
				return err
			}
			joinCommand = output
			break
		}
	}

	// now let the nodes join

	for _, node := range cluster.Nodes {
		if !node.IsMaster {
			_, err := runCmd(node, "swapoff -a && "+joinCommand)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func validateClusterCreateFlags(cmd *cobra.Command, args []string) error {

	var (
		name, ssh_key, master_server_type, worker_server_type string
	)
	if name, _ = cmd.Flags().GetString("name"); name == "" {
		return errors.New("flag --name is required")
	}

	if ssh_key, _ = cmd.Flags().GetString("ssh-key"); ssh_key == "" {
		return errors.New("flag --ssh-key is required")
	}

	if master_server_type, _ = cmd.Flags().GetString("master-server-type"); master_server_type == "" {
		return errors.New("flag --master_server_type is required")
	}

	if worker_server_type, _ = cmd.Flags().GetString("worker-server-type"); worker_server_type == "" {
		return errors.New("flag --worker_server_type is required")
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
}
