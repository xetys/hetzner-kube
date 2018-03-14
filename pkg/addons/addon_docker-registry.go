package addons

import (
	"log"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

type DockerregistryAddon struct {
	masterNode *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

func NewDockerregistryAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, err := provider.GetMasterNode()
	FatalOnError(err)
	return &DockerregistryAddon{masterNode: masterNode, communicator: communicator}
}

func (addon *DockerregistryAddon) Install(args ... string) {
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
