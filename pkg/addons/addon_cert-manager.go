package addons

import (
	"log"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

type CertmanagerAddon struct {
	masterNode *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

func NewCertmanagerAddon(cluster clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, err := cluster.GetMasterNode()
	FatalOnError(err)
	return &CertmanagerAddon{masterNode: masterNode, communicator:communicator}
}

func (addon *CertmanagerAddon) Install(args ...string) {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm install --name cert-manager --namespace ingress stable/cert-manager")
	FatalOnError(err)
	log.Println("cert-manager installed")
}

func (addon *CertmanagerAddon) Uninstall() {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm delete --purge cert-manager")
	FatalOnError(err)
	log.Println("cert-manager uninstalled")
}
