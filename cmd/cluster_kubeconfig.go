package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
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
	PreRunE: validateClusterInArgumentExists,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		_, cluster := AppConf.Config.FindClusterByName(name)

		provider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
		masterNode, err := provider.GetMasterNode()
		FatalOnError(err)

		err = AppConf.SSHClient.(*clustermanager.SSHCommunicator).CapturePassphrase(masterNode.SSHKeyName)
		FatalOnError(err)

		kubeConfigContent, err := AppConf.SSHClient.RunCmd(*masterNode, "cat /etc/rancher/rke2/rke2.yaml")
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
			if _, err := os.Stat(kubeconfigPath); !force && err == nil {
				fmt.Println("There already exists a kubeconfig. Overwrite? (use -f to suppress this question) [yN]:")
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

func init() {
	clusterCmd.AddCommand(clusterKubeconfigCmd)

	clusterKubeconfigCmd.Flags().StringP("name", "n", "", "name of the cluster")
	clusterKubeconfigCmd.Flags().BoolP("print", "p", false, "prints output to stdout")
	clusterKubeconfigCmd.Flags().BoolP("backup", "b", false, "saves existing config")
	clusterKubeconfigCmd.Flags().BoolP("force", "f", false, "don't ask to overwrite")
}
