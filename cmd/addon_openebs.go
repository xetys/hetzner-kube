package cmd

import "fmt"

type OpenEBSAddon struct {
	masterNode *Node
}

func NewOpenEBSAddon(cluster Cluster) ClusterAddon {
	masterNode, _ := cluster.GetMasterNode()
	return OpenEBSAddon{masterNode: masterNode}
}

func (addon OpenEBSAddon) Install(args ...string) {
	node := *addon.masterNode

	_, err := runCmd(node, "kubectl apply -f https://raw.githubusercontent.com/openebs/openebs/master/k8s/openebs-operator.yaml")
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
	err = writeNodeFile(node, "/root/openebs-storageclass.yaml", improvedStorageClass, false)
	FatalOnError(err)

	_, err = runCmd(node, "kubectl delete -f openebs-storageclass.yaml && kubectl apply -f openebs-storageclass.yaml")

	fmt.Println("OpenEBS installed")
}

func (addon OpenEBSAddon) Uninstall() {
	node := *addon.masterNode

	_, err := runCmd(node, "kubectl delete -f https://raw.githubusercontent.com/openebs/openebs/master/k8s/openebs-operator.yaml")
	FatalOnError(err)

	fmt.Println("OpenEBS removed")
}
