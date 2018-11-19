package addons

import (
	"fmt"

	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"io/ioutil"
	"log"
	"time"
)

//ScriptRunnerAddon installs script runner
type ScriptRunnerAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
	nodes        []clustermanager.Node
}

//NewScriptRunnerAddon installs script runner to the cluster
func NewScriptRunnerAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return ScriptRunnerAddon{masterNode: masterNode, communicator: communicator, nodes: provider.GetAllNodes()}
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

	var output string
	for _, node := range addon.nodes {
		scriptRemotePath := "/tmp/script-" + time.Now().Format("20060102150405") + ".sh"
		err = addon.communicator.WriteFile(node, scriptRemotePath, string(scriptContents), true)
		FatalOnError(err)

		output, err = addon.communicator.RunCmd(node, "bash "+scriptRemotePath)
		FatalOnError(err)
	}

	fmt.Println("Script ran successfully:\n", output)
}

//Uninstall performs all steps to remove the addon
func (addon ScriptRunnerAddon) Uninstall() {
	fmt.Println("no uninstall for this addon")
}
