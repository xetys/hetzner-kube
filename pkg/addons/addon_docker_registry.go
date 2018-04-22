package addons

import (
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"log"
)

type DockerregistryAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

func NewDockerregistryAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, err := provider.GetMasterNode()
	FatalOnError(err)
	return &DockerregistryAddon{masterNode: masterNode, communicator: communicator}
}

func init() {
	addAddon(NewDockerregistryAddon)
}

func (addon *DockerregistryAddon) Name() string {
	return "docker-registry"
}

func (addon *DockerregistryAddon) Description() string {
	return "Private container registry"
}

func (addon *DockerregistryAddon) URL() string {
	return ""
}

func (addon *DockerregistryAddon) Install(args ...string) {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm install --set persistence.enabled=true stable/docker-registry")
	FatalOnError(err)
	log.Println("docker-registry installed")
}

func (addon DockerregistryAddon) Uninstall() {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm delete --purge docker-registry")
	FatalOnError(err)
	log.Println("docker-registry uninstalled")
}
