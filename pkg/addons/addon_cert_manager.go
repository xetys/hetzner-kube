package addons

import (
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"log"
)

type CertmanagerAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

func NewCertmanagerAddon(cluster clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, err := cluster.GetMasterNode()
	FatalOnError(err)
	return &CertmanagerAddon{masterNode: masterNode, communicator: communicator}
}

func init() {
	addAddon(NewCertmanagerAddon)
}

func (addon *CertmanagerAddon) Name() string {
	return "cert-manager"
}

func (addon *CertmanagerAddon) Description() string {
	return "Auto-TLS provisioning & management"
}

func (addon *CertmanagerAddon) URL() string {
	return "https://github.com/jetstack/cert-manager"
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
