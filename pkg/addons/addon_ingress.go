package addons

import (
	"log"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

type IngressAddon struct {
	masterNode *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

func NewIngressAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, err := provider.GetMasterNode()
	FatalOnError(err)
	return &IngressAddon{masterNode: masterNode, communicator: communicator}
}

func (addon *IngressAddon) Install(args ...string) {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm install --name ingress --namespace ingress --set rbac.create=true,controller.kind=DaemonSet,controller.service.type=ClusterIP,controller.hostNetwork=true stable/nginx-ingress")
	FatalOnError(err)
	log.Println("nginx ingress installed")
}

func (addon *IngressAddon) Uninstall() {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm delete --purge ingress")
	FatalOnError(err)
	log.Println("nginx ingress uninstalled")
}
