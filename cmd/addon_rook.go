package cmd

import (
	"time"
	"fmt"
)

type RookAddon struct {
	masterNode *Node
}

func NewRookAddon(cluster Cluster) ClusterAddon {
	masterNode, _ := cluster.GetMasterNode()
	return RookAddon{masterNode: masterNode}
}

func (addon RookAddon) Install(args ... string) {
	node := *addon.masterNode

	_, err := runCmd(node, "kubectl apply -f https://github.com/rook/rook/raw/master/cluster/examples/kubernetes/rook-operator.yaml")
	FatalOnError(err)
	time.Sleep(15 * time.Second)
	_, err = runCmd(node, "kubectl apply -f https://github.com/rook/rook/raw/master/cluster/examples/kubernetes/rook-cluster.yaml")
	FatalOnError(err)
	_, err = runCmd(node, "kubectl apply -f https://github.com/rook/rook/raw/master/cluster/examples/kubernetes/rook-storageclass.yaml")
	FatalOnError(err)

	fmt.Println("Rook installed")
}

func (addon RookAddon) Uninstall() {
	node := *addon.masterNode
	_, err := runCmd(node, "kubectl delete -n rook pool replicapool")
	FatalOnError(err)
	_, err = runCmd(node, "kubectl delete storageclass rook-block")
	FatalOnError(err)
	_, err = runCmd(node, "kubectl delete -n kube-system secret rook-admin")
	FatalOnError(err)
	_, err = runCmd(node, "kubectl delete thirdpartyresources cluster.rook.io pool.rook.io objectstore.rook.io filesystem.rook.io volumeattachment.rook.io # ignore errors if on K8s 1.7+")
	FatalOnError(err)
	_, err = runCmd(node, "kubectl delete crd clusters.rook.io pools.rook.io objectstores.rook.io filesystems.rook.io volumeattachments.rook.io  # ignore errors if on K8s 1.5 and 1.6")
	FatalOnError(err)
	_, err = runCmd(node, "kubectl delete -n rook-system daemonset rook-agent")
	FatalOnError(err)
	_, err = runCmd(node, "kubectl delete -f rook-operator.yaml")
	FatalOnError(err)
	_, err = runCmd(node, "kubectl delete clusterroles rook-agent")
	FatalOnError(err)
	_, err = runCmd(node, "kubectl delete clusterrolebindings rook-agent")
	FatalOnError(err)
	_, err = runCmd(node, "kubectl delete namespace rook")
	FatalOnError(err)

	fmt.Println("Rook uninstalled")
}
