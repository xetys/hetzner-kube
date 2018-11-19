package addons

import (
	"fmt"

	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

//UfwAddon installs ufw
type UfwAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
	nodeCidr     string
	nodes        []clustermanager.Node
}

//NewUfwAddon installs ufw to the cluster
func NewUfwAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return UfwAddon{masterNode: masterNode, communicator: communicator, nodeCidr: provider.GetNodeCidr(), nodes: provider.GetAllNodes()}
}

func init() {
	addAddon(NewUfwAddon)
}

//Name returns the addons name
func (addon UfwAddon) Name() string {
	return "ufw"
}

//Requires returns a slice with the name of required addons
func (addon UfwAddon) Requires() []string {
	return []string{}
}

//Description returns the addons description
func (addon UfwAddon) Description() string {
	return "Uncomplicated Firewall"
}

//URL returns the URL of the addons underlying project
func (addon UfwAddon) URL() string {
	return "https://wiki.ubuntu.com/UncomplicatedFirewall"
}

//Install performs all steps to install the addon
func (addon UfwAddon) Install(args ...string) {

	var nodeIpRules string
	for _, node := range addon.nodes {
		nodeIpRules += " && ufw allow in from " + node.IPAddress + " to any"
	}
	var output string
	for _, node := range addon.nodes {
		_, err := addon.communicator.RunCmd(node, "apt-get install -y ufw")
		FatalOnError(err)

		fmt.Println("ufw installed on " + node.Name)

		_, err = addon.communicator.RunCmd(
			node,
			"ufw --force reset"+
				nodeIpRules+
				" && ufw allow ssh"+
				" && ufw allow in from "+addon.nodeCidr+" to any"+ // Kubernetes VPN overlay interface
				" && ufw allow in from 10.244.0.0/16 to any"+ // Kubernetes pod overlay interface
				" && ufw allow 6443"+ // Kubernetes API secure remote port
				" && ufw allow 80"+
				" && ufw allow 443"+
				" && ufw default deny incoming"+
				" && ufw --force enable")
		FatalOnError(err)

		output, err = addon.communicator.RunCmd(node, "ufw status verbose")
		FatalOnError(err)
	}

	fmt.Println("ufw enabled with the following rules:\n", output)
}

//Uninstall performs all steps to remove the addon
func (addon UfwAddon) Uninstall() {
	for _, node := range addon.nodes {
		_, err := addon.communicator.RunCmd(node, "ufw --force reset && ufw --force disable")
		FatalOnError(err)
	}
	fmt.Println("ufw uninstalled")
}
