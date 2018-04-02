package addons

import (
	"fmt"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"time"
)

type RookAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
	nodes        []clustermanager.Node
}

func NewRookAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return &RookAddon{masterNode: masterNode, communicator: communicator, nodes: provider.GetAllNodes()}
}

func (addon RookAddon) Install(args ...string) {
	node := *addon.masterNode

	_, err := addon.communicator.RunCmd(node, "kubectl apply -f https://github.com/rook/rook/raw/master/cluster/examples/kubernetes/rook-operator.yaml")
	FatalOnError(err)
	fmt.Println("waiting until rook is installed")
	for {
		_, err := addon.communicator.RunCmd(node, "kubectl get cluster")

		if err == nil {
			break
		}
	}
	_, err = addon.communicator.RunCmd(node, "kubectl apply -f https://github.com/rook/rook/raw/master/cluster/examples/kubernetes/rook-cluster.yaml")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl apply -f https://github.com/rook/rook/raw/master/cluster/examples/kubernetes/rook-storageclass.yaml")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl apply -f https://github.com/rook/rook/raw/master/cluster/examples/kubernetes/rook-tools.yaml")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl get storageclass | grep -v 'NAME' | awk '{print$1}' | xargs kubectl patch storageclass -p '{\"metadata\": {\"annotations\":{\"storageclass.kubernetes.io/is-default-class\":\"false\"}}}'")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl patch storageclass rook-block -p '{\"metadata\": {\"annotations\":{\"storageclass.kubernetes.io/is-default-class\":\"true\"}}}'")
	FatalOnError(err)

	fmt.Println("Rook installed")
}

func (addon RookAddon) Uninstall() {
	node := *addon.masterNode
	addon.communicator.RunCmd(node, "kubectl delete -n rook pool replicapool")
	addon.communicator.RunCmd(node, "kubectl delete storageclass rook-block")
	addon.communicator.RunCmd(node, "kubectl delete crd clusters.rook.io pools.rook.io objectstores.rook.io filesystems.rook.io volumeattachments.rook.io  # ignore errors if on K8s 1.5 and 1.6")
	addon.communicator.RunCmd(node, "kubectl delete -n rook-system daemonset rook-agent")
	addon.communicator.RunCmd(node, "kubectl delete -f https://github.com/rook/rook/raw/master/cluster/examples/kubernetes/rook-operator.yaml")
	addon.communicator.RunCmd(node, "kubectl delete clusterroles rook-agent")
	addon.communicator.RunCmd(node, "kubectl delete clusterrolebindings rook-agent")
	time.Sleep(20 * time.Second)
	addon.communicator.RunCmd(node, "kubectl delete namespace rook")

	for _, node := range addon.nodes {
		fmt.Printf("deleting rook on node %s\n", node.Name)
		addon.communicator.RunCmd(node, "rm -rf /var/lib/rook")
	}

	fmt.Println("Rook uninstalled")
}
