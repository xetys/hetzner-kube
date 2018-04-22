package addons

import (
	"fmt"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

type OpenEBSAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

func NewOpenEBSAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return &OpenEBSAddon{masterNode: masterNode, communicator: communicator}
}

func init() {
	addAddon(NewOpenEBSAddon)
}

func (addon OpenEBSAddon) Name() string {
	return "OpenEBS"
}

func (addon OpenEBSAddon) Requires() []string {
	return []string{}
}

func (addon OpenEBSAddon) Description() string {
	return "Simple scalable block storage provider"
}

func (addon OpenEBSAddon) URL() string {
	return "https://openebs.io/"
}

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
	err = addon.communicator.WriteFile(node, "/root/openebs-storageclass.yaml", improvedStorageClass, false)
	FatalOnError(err)

	_, err = addon.communicator.RunCmd(node, "kubectl delete -f openebs-storageclass.yaml && kubectl apply -f openebs-storageclass.yaml")
	FatalOnError(err)

	fmt.Println("OpenEBS installed")
}

func (addon OpenEBSAddon) Uninstall() {
	node := *addon.masterNode

	_, err := addon.communicator.RunCmd(node, "kubectl delete -f https://raw.githubusercontent.com/openebs/openebs/master/k8s/openebs-operator.yaml")
	FatalOnError(err)

	fmt.Println("OpenEBS removed")
}
