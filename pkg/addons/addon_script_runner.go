package addons

import (
	"fmt"

	"encoding/json"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

//ScriptRunnerAddon installs script runner
type ScriptRunnerAddon struct {
	communicator clustermanager.NodeCommunicator
	nodes        []clustermanager.Node
	cluster      clustermanager.Cluster
}

//NewScriptRunnerAddon installs script runner to the cluster
func NewScriptRunnerAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	return ScriptRunnerAddon{communicator: communicator, nodes: provider.GetAllNodes(), cluster: provider.GetCluster()}
}

func init() {
	addAddon(NewScriptRunnerAddon)
}

//Name returns the addons name
func (addon ScriptRunnerAddon) Name() string {
	return "script-runner"
}

//Requires returns a slice with the name of required addons
func (addon ScriptRunnerAddon) Requires() []string {
	return []string{}
}

//Description returns the addons description
func (addon ScriptRunnerAddon) Description() string {
	return "Bash remote script runner"
}

//URL returns the URL of the addons underlying project
func (addon ScriptRunnerAddon) URL() string {
	return "https://www.gnu.org/software/bash/"
}

//Install performs all steps to install the addon
func (addon ScriptRunnerAddon) Install(args ...string) {

	if len(args) < 2 {
		log.Fatal("path argument is missing")
	}
	scriptPath := args[1]
	scriptContents, err := ioutil.ReadFile(scriptPath)
	FatalOnError(err)

	clusterInfoBin, err := json.Marshal(addon.cluster)
	FatalOnError(err)

	replacer := strings.NewReplacer("\n", "", "'", "\\'")
	clusterInfo := replacer.Replace(string(clusterInfoBin))

	for _, node := range addon.nodes {
		scriptRemotePath := "/tmp/script-" + time.Now().Format("20060102150405") + ".sh"
		err = addon.communicator.WriteFile(node, scriptRemotePath, string(scriptContents), true)
		FatalOnError(err)

		output, err := addon.communicator.RunCmd(
			node,
			"bash "+
				scriptRemotePath+
				" "+node.Group+
				" '"+clusterInfo+"'")
		FatalOnError(err)
		fmt.Println(node.Name+" "+node.IPAddress+": script ran successfully..\n", output)
	}
}

//Uninstall performs all steps to remove the addon
func (addon ScriptRunnerAddon) Uninstall() {
	fmt.Println("no uninstall for this addon")
}
