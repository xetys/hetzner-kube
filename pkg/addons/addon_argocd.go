package addons

import (
	"fmt"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

// ArgoCDAddon install ArgoCD in Cluster
type ArgoCDAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
	provider     *hetzner.Provider
}

// Name returns the addons' name
func (addon *ArgoCDAddon) Name() string {
	return "argocd"
}

// Requires returns the addons' requirements
func (addon *ArgoCDAddon) Requires() []string {
	return []string{}
}

// Description returns the addons' description
func (addon *ArgoCDAddon) Description() string {
	return "ArgoCD integration"
}

// URL returns the hcloud-csi URL
func (addon *ArgoCDAddon) URL() string {
	return "https://argoproj.github.io/argo-cd/"
}

// Install argocd and shows the login-token
func (addon *ArgoCDAddon) Install(args ...string) {
	// add namespace
	_, err := addon.communicator.RunCmd(*addon.masterNode, "kubectl create namespace argocd")
	FatalOnError(err)

	// add argocd
	_, err = addon.communicator.RunCmd(*addon.masterNode, "kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml")
	FatalOnError(err)

	// Get token for login
	token, err := addon.communicator.RunCmd(*addon.masterNode, "kubectl get pods -n argocd -l app.kubernetes.io/name=argocd-server -o name | cut -d'/' -f 2")
	FatalOnError(err)

	fmt.Printf("Use the following token to login into argocd: %s\n", token)
	fmt.Println("ArgoCD installed")
}

// Uninstall performs the reverse steps of Install
func (addon *ArgoCDAddon) Uninstall() {
	// delete hcloud-csi
	_, err := addon.communicator.RunCmd(*addon.masterNode, "kubectl delete -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml --ignore-not-found")
	if err != nil {
		FatalOnError(err)
	}

	// delete csi driver
	_, err = addon.communicator.RunCmd(*addon.masterNode, "kubectl delete namespace argocd --ignore-not-found")
	if err != nil {
		FatalOnError(err)
	}
}

// NewArgoCDAddon creates an instance of ArgoCDAddon
func NewArgoCDAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return &ArgoCDAddon{
		masterNode:   masterNode,
		communicator: communicator,
		provider:     provider.(*hetzner.Provider),
	}
}

// adding the addon to the global list
func init() {
	addAddon(NewArgoCDAddon)
}
