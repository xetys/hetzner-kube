package addons

import (
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

// HetznerCSIAddon install CSI drivers and hcloud-csi integration to the cluster
type HetznerCSIAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
	provider     *hetzner.Provider
}

// Name returns the addons' name
func (addon *HetznerCSIAddon) Name() string {
	return "hetzner-csi"
}

// Requires returns the addons' requirements
func (addon *HetznerCSIAddon) Requires() []string {
	return []string{}
}

// Description returns the addons' description
func (addon *HetznerCSIAddon) Description() string {
	return "Hetzner CSI integration"
}

// URL returns the hcloud-csi URL
func (addon *HetznerCSIAddon) URL() string {
	return "https://github.com/hetznercloud/csi-driver"
}

// Install installs a token secret and the hcloud-cs components to the cluster
func (addon *HetznerCSIAddon) Install(args ...string) {
	// add secret
	_, err := addon.communicator.RunCmd(*addon.masterNode, "kubectl -n kube-system create secret generic hcloud-csi --from-literal=token="+addon.provider.Token())
	if err != nil {
		FatalOnError(err)
	}

	// add csi driver
	_, err = addon.communicator.RunCmd(*addon.masterNode, "kubectl apply -f https://raw.githubusercontent.com/kubernetes/csi-api/release-1.40/pkg/crd/manifests/csidriver.yaml")
	if err != nil {
		FatalOnError(err)
	}
	_, err = addon.communicator.RunCmd(*addon.masterNode, "kubectl apply -f https://raw.githubusercontent.com/kubernetes/csi-api/release-1.40/pkg/crd/manifests/csinodeinfo.yaml")
	if err != nil {
		FatalOnError(err)
	}

	// install hcloud-csi
	_, err = addon.communicator.RunCmd(*addon.masterNode, "kubectl apply -f https://raw.githubusercontent.com/hetznercloud/csi-driver/master/deploy/kubernetes/hcloud-csi.yml")
	if err != nil {
		FatalOnError(err)
	}
}

// Uninstall performs the reverse steps of Install
func (addon *HetznerCSIAddon) Uninstall() {
	// delete hcloud-csi
	_, err := addon.communicator.RunCmd(*addon.masterNode, "kubectl delete -f https://raw.githubusercontent.com/hetznercloud/csi-driver/master/deploy/kubernetes/hcloud-csi.yml --ignore-not-found")
	if err != nil {
		FatalOnError(err)
	}

	// delete csi driver
	_, err = addon.communicator.RunCmd(*addon.masterNode, "kubectl delete -f https://raw.githubusercontent.com/kubernetes/csi-api/release-1.40/pkg/crd/manifests/csidriver.yaml --ignore-not-found")
	if err != nil {
		FatalOnError(err)
	}
	_, err = addon.communicator.RunCmd(*addon.masterNode, "kubectl delete -f https://raw.githubusercontent.com/kubernetes/csi-api/release-1.40/pkg/crd/manifests/csinodeinfo.yaml --ignore-not-found")
	if err != nil {
		FatalOnError(err)
	}

	// delete secret
	_, err = addon.communicator.RunCmd(*addon.masterNode, "kubectl -n kube-system delete secret hcloud-csi --ignore-not-found")
	if err != nil {
		FatalOnError(err)
	}

}

// NewHetznerCSIAddon creates an instance of HetznerCSIAddon
func NewHetznerCSIAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return &HetznerCSIAddon{
		masterNode:   masterNode,
		communicator: communicator,
		provider:     provider.(*hetzner.Provider),
	}
}

// adding the addon to the global list
func init() {
	addAddon(NewHetznerCSIAddon)
}
