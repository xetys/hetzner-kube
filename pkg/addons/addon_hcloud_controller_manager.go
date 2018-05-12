package addons

import (
	"fmt"

	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

// HCloudControllerManagerAddon installs hetzner clouds official cloud controller manager
type HCloudControllerManagerAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
	nodes        []clustermanager.Node
	provider     *hetzner.Provider
}

// NewHCloudControllerManagerAddon returns a CloudProvider instance with type HCloudControllerManagerAddon
func NewHCloudControllerManagerAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, err := provider.GetMasterNode()
	FatalOnError(err)
	return &HCloudControllerManagerAddon{
		masterNode:   masterNode,
		communicator: communicator,
		nodes:        provider.GetAllNodes(),
		provider:     provider.(*hetzner.Provider),
	}
}

func init() {
	addAddon(NewHCloudControllerManagerAddon)
}

//Name returns the addons name
func (addon *HCloudControllerManagerAddon) Name() string {
	return "hcloud-controller-manager"
}

//Requires returns a slice with the name of required addons
func (addon *HCloudControllerManagerAddon) Requires() []string {
	return []string{}
}

//Description returns the addons description
func (addon *HCloudControllerManagerAddon) Description() string {
	return "Hetzner Cloud official cloud controller manager"
}

//URL returns the URL of the addons underlying project
func (addon *HCloudControllerManagerAddon) URL() string {
	return "https://github.com/hetznercloud/hcloud-cloud-controller-manager"
}

//Install performs all steps to install the addon
func (addon *HCloudControllerManagerAddon) Install(args ...string) {
	// set external cloud provider
	config := `
[Service]
Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
`
	for _, node := range addon.nodes {
		err := addon.communicator.WriteFile(node, "/etc/systemd/system/kubelet.service.d/20-hcloud.conf", config, false)
		FatalOnError(err)

		_, err = addon.communicator.RunCmd(node, "systemctl daemon-reload && systemctl restart kubelet")
		FatalOnError(err)
	}

	_, err := addon.communicator.RunCmd(*addon.masterNode, `kubectl -n kube-system patch ds kube-flannel-ds --type json -p '[{"op":"add","path":"/spec/template/spec/tolerations/-","value":{"key":"node.cloudprovider.kubernetes.io/uninitialized","value":"true","effect":"NoSchedule"}}]'`)
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(*addon.masterNode, fmt.Sprintf("kubectl -n kube-system create secret generic hcloud --from-literal=token=%s", addon.provider.Token()))
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(*addon.masterNode, "kubectl apply -f  https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/v1.2.0.yaml")
	FatalOnError(err)
}

//Uninstall performs all steps to remove the addon
func (addon *HCloudControllerManagerAddon) Uninstall() {
	_, err := addon.communicator.RunCmd(*addon.masterNode, "kubectl delete -f  https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/v1.2.0.yaml")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(*addon.masterNode, "kubectl -n kube-system delete secret hcloud")
	FatalOnError(err)

	for _, node := range addon.nodes {
		_, err := addon.communicator.RunCmd(node, "rm /etc/systemd/system/kubelet.service.d/20-hcloud.conf")
		FatalOnError(err)

		_, err = addon.communicator.RunCmd(node, "systemctl daemon-reload && systemctl restart kubelet")
		FatalOnError(err)
	}
}
