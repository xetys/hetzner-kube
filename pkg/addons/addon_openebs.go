package addons

import (
	"fmt"

	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

// OpenEBSAddon installs OpenEBS
type OpenEBSAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

// NewOpenEBSAddon creates an addon which installs OpenEBS
func NewOpenEBSAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return &OpenEBSAddon{masterNode: masterNode, communicator: communicator}
}

// Name returns the addons name
func (addon OpenEBSAddon) Name() string {
	return "openebs"
}

// Requires returns a slice with the name of required addons
func (addon OpenEBSAddon) Requires() []string {
	return []string{}
}

// Description returns the addons description
func (addon OpenEBSAddon) Description() string {
	return "Simple scalable block storage provider"
}

// URL returns the URL of the addons underlying project
func (addon OpenEBSAddon) URL() string {
	return "https://openebs.io/"
}

// Install performs all steps to install the addon
func (addon OpenEBSAddon) Install(args ...string) {
	node := *addon.masterNode

	_, err := addon.communicator.RunCmd(node, "kubectl apply -f https://raw.githubusercontent.com/openebs/openebs/master/k8s/openebs-operator.yaml")
	FatalOnError(err)

	improvedStorageClass := `apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: openebs-standard
provisioner: openebs.io/provisioner-iscsi
parameters:
  openebs.io/storage-pool: "default"
  openebs.io/jiva-replica-count: "3"
  openebs.io/volume-monitor: "true"
  openebs.io/capacity: 5G`
	err = addon.communicator.WriteFile(node, "/root/openebs-storageclass.yaml", improvedStorageClass, clustermanager.AllRead)
	FatalOnError(err)

	_, err = addon.communicator.RunCmd(node, "kubectl delete -f openebs-storageclass.yaml ; kubectl apply -f openebs-storageclass.yaml")
	FatalOnError(err)

	fmt.Println("OpenEBS installed")
}

// Uninstall performs all steps to remove the addon
func (addon OpenEBSAddon) Uninstall() {
	node := *addon.masterNode

	_, err := addon.communicator.RunCmd(node, "kubectl delete -f https://raw.githubusercontent.com/openebs/openebs/master/k8s/openebs-operator.yaml")
	FatalOnError(err)

	fmt.Println("OpenEBS removed")
}
