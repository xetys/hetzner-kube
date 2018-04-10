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
	"bufio"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

// clusterKubeconfigCmd represents the clusterKubeconfig command
var clusterKubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig <CLUSTER NAME>",
	Short: "setups the kubeconfig for the local machine",
	Long: `fetches the kubeconfig (e.g. for usage with kubectl) and saves it to ~/.kube/config, or prints it.

Example 1: hetzner-kube cluster kubeconfig -n my-cluster # installs the kubeconfig of the cluster "my-cluster"
Example 2: hetzner-kube cluster kubeconfig -n my-cluster -b # saves the existing before installing
Example 3: hetzner-kube cluster kubeconfig -n my-cluster -p # prints the contents of kubeconfig to console
Example 4: hetzner-kube cluster kubeconfig -n my-cluster -p > my-conf.yaml # prints the contents of kubeconfig into a custom file
	`,
	Args:    cobra.ExactArgs(1),
	PreRunE: validateKubeconfigCmd,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		_, cluster := AppConf.Config.FindClusterByName(name)

		provider, _ := hetzner.ProviderAndManager(*cluster, AppConf.Client, AppConf.Context, AppConf.SSHClient, nil, AppConf.CurrentContext.Token)
		masterNode, err := provider.GetMasterNode()
		FatalOnError(err)

		err = AppConf.SSHClient.(*clustermanager.SSHCommunicator).CapturePassphrase(masterNode.SSHKeyName)
		FatalOnError(err)

		kubeConfigContent, err := AppConf.SSHClient.RunCmd(*masterNode, "cat /etc/kubernetes/admin.conf")
		// change the IP to public
		kubeConfigContent = strings.Replace(kubeConfigContent, masterNode.PrivateIPAddress, masterNode.IPAddress, -1)

		FatalOnError(err)

		printContent, _ := cmd.Flags().GetBool("print")
		force, _ := cmd.Flags().GetBool("force")

		if printContent {
			fmt.Println(kubeConfigContent)
		} else {
			fmt.Println("create file")

			usr, _ := user.Current()
			dir := usr.HomeDir
			path := fmt.Sprintf("%s/.kube", dir)

			if _, err := os.Stat(path); os.IsNotExist(err) {
				os.MkdirAll(path, 0755)
			}

			// check if there already is an existing config
			kubeconfigPath := fmt.Sprintf("%s/config", path)
			if _, err := os.Stat(path); !force && err == nil {
				fmt.Println("There already exists a kubeconfig. Overwrite? (use -f to supress this question) [yN]:")
				r := bufio.NewReader(os.Stdin)
				answer, err := r.ReadString('\n')
				FatalOnError(err)
				if !strings.ContainsAny(answer, "yY") {
					log.Fatalln("aborted")
				}
			}

			ioutil.WriteFile(kubeconfigPath, []byte(kubeConfigContent), 0755)

			fmt.Println("kubeconfig configured")
		}
	},
}

func validateKubeconfigCmd(cmd *cobra.Command, args []string) error {

	name := args[0]

	if name == "" {
		return errors.New("flag --name is required")
	}

	idx, _ := AppConf.Config.FindClusterByName(name)

	if idx == -1 {
		return fmt.Errorf("cluster '%s' not found", name)
	}
	return nil
}

func init() {
	clusterCmd.AddCommand(clusterKubeconfigCmd)

	clusterKubeconfigCmd.Flags().StringP("name", "n", "", "name of the cluster")
	clusterKubeconfigCmd.Flags().BoolP("print", "p", false, "prints output to stdout")
	clusterKubeconfigCmd.Flags().BoolP("backup", "b", false, "saves existing config")
	clusterKubeconfigCmd.Flags().BoolP("force", "f", false, "don't ask to overwrite")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clusterKubeconfigCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clusterKubeconfigCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
