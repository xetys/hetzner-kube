package addons

import (
	"fmt"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"time"
)

type RookAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

func NewRookAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return &RookAddon{masterNode: masterNode, communicator: communicator}
}

func (addon RookAddon) Install(args ...string) {
	node := *addon.masterNode

	_, err := addon.communicator.RunCmd(node, "kubectl apply -f https://github.com/rook/rook/raw/master/cluster/examples/kubernetes/rook-operator.yaml")
	FatalOnError(err)
	time.Sleep(15 * time.Second)
	_, err = addon.communicator.RunCmd(node, "kubectl apply -f https://github.com/rook/rook/raw/master/cluster/examples/kubernetes/rook-cluster.yaml")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl apply -f https://github.com/rook/rook/raw/master/cluster/examples/kubernetes/rook-storageclass.yaml")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl get storageclass | grep -v 'NAME' | awk '{print$1}' | xargs kubectl patch storageclass -p '{\"metadata\": {\"annotations\":{\"storageclass.kubernetes.io/is-default-class\":\"false\"}}}'")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl patch storageclass rook-block -p '{\"metadata\": {\"annotations\":{\"storageclass.kubernetes.io/is-default-class\":\"true\"}}}'")
	FatalOnError(err)

	fmt.Println("Rook installed")
}

func (addon RookAddon) Uninstall() {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "kubectl delete -n rook pool replicapool")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl delete storageclass rook-block")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl delete -n kube-system secret rook-admin")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl delete thirdpartyresources cluster.rook.io pool.rook.io objectstore.rook.io filesystem.rook.io volumeattachment.rook.io # ignore errors if on K8s 1.7+")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl delete crd clusters.rook.io pools.rook.io objectstores.rook.io filesystems.rook.io volumeattachments.rook.io  # ignore errors if on K8s 1.5 and 1.6")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl delete -n rook-system daemonset rook-agent")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl delete -f rook-operator.yaml")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl delete clusterroles rook-agent")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl delete clusterrolebindings rook-agent")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl delete namespace rook")
	FatalOnError(err)

	fmt.Println("Rook uninstalled")
}
