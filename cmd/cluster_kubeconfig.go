package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	default_context = "kubernetes-admin@kubernetes"
)

// clusterKubeconfigCmd represents the clusterKubeconfig command
var clusterKubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig <CLUSTER NAME>",
	Short: "setups the kubeconfig for the local machine",
	Long: `fetches the kubeconfig (e.g. for usage with kubectl) and saves it to ~/.kube/config, or prints it.

Example 1: hetzner-kube cluster kubeconfig my-cluster                        # prints the kubeconfig of the cluster "my-cluster"
Example 2: hetzner-kube cluster kubeconfig my-cluster > my-conf.yaml         # prints the contents of kubeconfig into a custom file
Example 3: hetzner-kube cluster kubeconfig my-cluster -s -t ./my-conf.yaml   # saves the contents of kubeconfig into a custom file
Example 4: hetzner-kube cluster kubeconfig my-cluster -m                     # merges the existing with current cluster (creates backup before merge)
    `,
	Args:    cobra.ExactArgs(1),
	PreRunE: validateKubeconfigCmd,
	Run: func(cmd *cobra.Command, args []string) {

		name := args[0]
		_, cluster := AppConf.Config.FindClusterByName(name)

		provider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
		masterNode, err := provider.GetMasterNode()
		FatalOnError(err)

		err = AppConf.SSHClient.(*clustermanager.SSHCommunicator).CapturePassphrase(masterNode.SSHKeyName)
		FatalOnError(err)

		kubeConfigContent, err := AppConf.SSHClient.RunCmd(*masterNode, "cat /etc/kubernetes/admin.conf")
		// change the IP to public
		kubeConfigContent = strings.Replace(kubeConfigContent, masterNode.PrivateIPAddress, masterNode.IPAddress, -1)

		FatalOnError(err)

		// get sanitized kubeconfig
		// we need is_sanitized flag to ensure we want do a merge if this it fails
		is_sanitized := false
		newKubeConfig, err := sanitizeKubeConfig(kubeConfigContent, provider.GetCluster().Name, "hetzner")
		if err != nil {
			log.Printf("KubeConfig sanitise process failed, default config will be used instead. Error: %s", err.Error())
		} else {
			kubeConfigContent = newKubeConfig
			is_sanitized = true
		}

		if merge, _ := cmd.Flags().GetBool("merge"); merge && is_sanitized {

			kubeConfigPath := fmt.Sprintf("%s/.kube/config", GetHome())
			if _, err := os.Stat(kubeConfigPath); err == nil {
				doConfigCopy(kubeConfigPath)
			}
			return
		}

		if save, _ := cmd.Flags().GetBool("save"); save {

			target_path := fmt.Sprintf("%s/.kube/%s.yaml", GetHome(), provider.GetCluster().Name)
			if target, _ := cmd.Flags().GetString("target"); target != "" {
				target_path = target
			}
			log.Printf("Saving current config to '%s'", target_path)
			doConfigWrite(target_path, kubeConfigContent)

			return
		}

		fmt.Println(kubeConfigContent)
	},
}

// Write kubeConfig to destination
func doConfigWrite(dst string, kubeConfig string) (err error) {

	if _, err := os.Stat(path.Dir(dst)); os.IsNotExist(err) {
		os.MkdirAll(path.Dir(dst), 0755)
	}
	return ioutil.WriteFile(dst, []byte(kubeConfig), 0755)
}

// Create backup of current kubeCongig
func doConfigCopy(src string) (err error) {
	var source, destination *os.File
	if source, err = os.Open(src); err != nil {
		return
	}
	defer source.Close()

	dst := fmt.Sprintf("%s/config.%s", path.Dir(src), time.Now().Format("20060102150405"))
	if destination, err = os.Create(dst); err != nil {
		return
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	log.Printf("KubeConfig backup save as '%s'", dst)
	return
}

func sanitizeKubeConfig(kubeConfig string, clusterName string, prefix string) (string, error) {

	// Read kubeconfig to k8s config structure
	apiCfg, err := clientcmd.Load([]byte(kubeConfig))
	if err != nil {
		return "", err
	}

	// get our default Context from configuration (check `const` section)
	var ctx *clientcmdapi.Context
	if ctx = apiCfg.Contexts[default_context]; ctx == nil {
		return "", errors.New(fmt.Sprintf("Default context '%s' does not found in current configuration!", default_context))
	}

	// Apply prefix if it set
	if prefix != "" {
		clusterName = fmt.Sprintf("%s-%s", prefix, clusterName)
	}

	// save current cluster name and authInfo Names
	current_cluster := ctx.Cluster
	current_authInfo := ctx.AuthInfo

	// define new Cluster and AuthInfo Names as Project Name
	ctx.Cluster = clusterName
	ctx.AuthInfo = clusterName

	// Copy current data about Context,Cluster,authInfo with new Names
	apiCfg.Contexts[clusterName] = ctx
	apiCfg.Clusters[clusterName] = apiCfg.Clusters[current_cluster]
	apiCfg.AuthInfos[clusterName] = apiCfg.AuthInfos[current_authInfo]
	apiCfg.CurrentContext = clusterName

	// Remove outdaited details
	delete(apiCfg.Clusters, current_cluster)
	delete(apiCfg.AuthInfos, current_authInfo)
	delete(apiCfg.Contexts, default_context)

	configByte, err := clientcmd.Write(*apiCfg)
	if err != nil {
		return "", err
	}
	return string(configByte), nil
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

	clusterKubeconfigCmd.Flags().BoolP("merge", "m", false, "merges .kube/config with my-cluster config")
	clusterKubeconfigCmd.Flags().BoolP("save", "s", false, "saves current config to target location, requires set `--target| -t`")
	clusterKubeconfigCmd.Flags().StringP("target", "t", "", "saves current config to target location (if not set, default to ~/.kube/my-cluster-config)")
}
