package addons

import (
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"log"
)

type DockerregistryAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

// NewDockerregistryAddon creates an addon providing a private docker registry
func NewDockerregistryAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, err := provider.GetMasterNode()
	FatalOnError(err)
	return &DockerregistryAddon{masterNode: masterNode, communicator: communicator}
}

func init() {
	addAddon(NewDockerregistryAddon)
}

//Name returns the addons name
func (addon *DockerregistryAddon) Name() string {
	return "docker-registry"
}

//Requires returns a slice with the name of required addons
func (addon *DockerregistryAddon) Requires() []string {
	return []string{"helm"}
}

//Description returns the addons description
func (addon *DockerregistryAddon) Description() string {
	return "Private container registry"
}

//URL returns the URL of the addons underlying project
func (addon *DockerregistryAddon) URL() string {
	return "https://github.com/kubernetes/charts/tree/master/stable/docker-registry"
}

//Install performs all steps to install the addon
func (addon *DockerregistryAddon) Install(args ...string) {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm install --set persistence.enabled=true stable/docker-registry")
	FatalOnError(err)
	log.Println("docker-registry installed")
}

//Uninstall performs all steps to remove the addon
func (addon DockerregistryAddon) Uninstall() {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm delete --purge docker-registry")
	FatalOnError(err)
	log.Println("docker-registry uninstalled")
}
