package addons

import (
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"log"
)

//IngressAddon installs an ingress controller
type IngressAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

//NewIngressAddon creates an addon to install a nginx ingress controller
func NewIngressAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, err := provider.GetMasterNode()
	FatalOnError(err)
	return &IngressAddon{masterNode: masterNode, communicator: communicator}
}

func init() {
	addAddon(NewIngressAddon)
}

//Name returns the addons name
func (addon *IngressAddon) Name() string {
	return "nginx-ingress-controller"
}

//Requires returns a slice with the name of required addons
func (addon *IngressAddon) Requires() []string {
	return []string{"helm"}
}

//Description returns the addons description
func (addon *IngressAddon) Description() string {
	return "an ingress based load balancer for K8S"
}

//URL returns the URL of the addons underlying project
func (addon *IngressAddon) URL() string {
	return ""
}

//Install performs all steps to install the addon
func (addon *IngressAddon) Install(args ...string) {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm install --name ingress --namespace ingress --set rbac.create=true,controller.kind=DaemonSet,controller.service.type=ClusterIP,controller.hostNetwork=true stable/nginx-ingress")
	FatalOnError(err)
	log.Println("nginx ingress installed")
}

//Uninstall performs all steps to remove the addon
func (addon *IngressAddon) Uninstall() {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm delete --purge ingress")
	FatalOnError(err)
	log.Println("nginx ingress uninstalled")
}
