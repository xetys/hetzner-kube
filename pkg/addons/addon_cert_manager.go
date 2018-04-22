package addons

import (
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"log"
)

//CertmanagerAddon installs cert-manager
type CertmanagerAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

// NewCertmanagerAddon creates an addon installing cert-manager
func NewCertmanagerAddon(cluster clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, err := cluster.GetMasterNode()
	FatalOnError(err)
	return &CertmanagerAddon{masterNode: masterNode, communicator: communicator}
}

func init() {
	addAddon(NewCertmanagerAddon)
}

//Name returns the addons name
func (addon *CertmanagerAddon) Name() string {
	return "cert-manager"
}

//Requires returns a slice with the name of required addons
func (addon *CertmanagerAddon) Requires() []string {
	return []string{"helm"}
}

//Description returns the addons description
func (addon *CertmanagerAddon) Description() string {
	return "Auto-TLS provisioning & management"
}

//URL returns the URL of the addons underlying project
func (addon *CertmanagerAddon) URL() string {
	return "https://github.com/jetstack/cert-manager"
}

//Install performs all steps to install the addon
func (addon *CertmanagerAddon) Install(args ...string) {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm install --name cert-manager --namespace ingress stable/cert-manager")
	FatalOnError(err)
	log.Println("cert-manager installed")
}

//Uninstall performs all steps to remove the addon
func (addon *CertmanagerAddon) Uninstall() {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm delete --purge cert-manager")
	FatalOnError(err)
	log.Println("cert-manager uninstalled")
}
