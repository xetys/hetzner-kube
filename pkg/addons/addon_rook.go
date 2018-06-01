package addons

import (
	"fmt"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"time"
)

//RookAddon installs rook
type RookAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
	nodes        []clustermanager.Node
}

//NewRookAddon creates an addon which install rook
func NewRookAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return &RookAddon{masterNode: masterNode, communicator: communicator, nodes: provider.GetAllNodes()}
}

func init() {
	addAddon(NewRookAddon)
}

//Name returns the addons name
func (addon RookAddon) Name() string {
	return "rook"
}

//Requires returns a slice with the name of required addons
func (addon RookAddon) Requires() []string {
	return []string{}
}

//Description returns the addons description
func (addon RookAddon) Description() string {
	return "File, Block and Object Storage provider"
}

//URL returns the URL of the addons underlying project
func (addon RookAddon) URL() string {
	return "https://rook.io"
}

//Install performs all steps to install the addon
func (addon RookAddon) Install(args ...string) {
	node := *addon.masterNode

	_, err := addon.communicator.RunCmd(node, "kubectl apply -f https://raw.githubusercontent.com/rook/rook/master/cluster/examples/kubernetes/ceph/operator.yaml")
	FatalOnError(err)
	fmt.Println("waiting until rook is installed")
	for {
		_, err := addon.communicator.RunCmd(node, "kubectl get cluster")

		if err == nil {
			break
		}
	}
	_, err = addon.communicator.RunCmd(node, "kubectl apply -f https://raw.github.com/rook/rook/master/cluster/examples/kubernetes/ceph/cluster.yaml")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl apply -f https://raw.github.com/rook/rook/master/cluster/examples/kubernetes/ceph/storageclass.yaml")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl apply -f https://raw.github.com/rook/rook/master/cluster/examples/kubernetes/ceph/toolbox.yaml")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl get storageclass | grep -v 'NAME' | awk '{print$1}' | xargs kubectl patch storageclass -p '{\"metadata\": {\"annotations\":{\"storageclass.kubernetes.io/is-default-class\":\"false\"}}}'")
	FatalOnError(err)
	_, err = addon.communicator.RunCmd(node, "kubectl patch storageclass rook-ceph-block -p '{\"metadata\": {\"annotations\":{\"storageclass.kubernetes.io/is-default-class\":\"true\"}}}'")
	FatalOnError(err)

	fmt.Println("Rook installed")
}

//Uninstall performs all steps to remove the addon
func (addon RookAddon) Uninstall() {
	node := *addon.masterNode
	addon.communicator.RunCmd(node, "kubectl delete -n rook pool replicapool")
	addon.communicator.RunCmd(node, "kubectl delete storageclass rook-ceph-block")
	addon.communicator.RunCmd(node, "kubectl delete crd clusters.rook.io pools.rook.io objectstores.rook.io filesystems.rook.io volumeattachments.rook.io  # ignore errors if on K8s 1.5 and 1.6")
	addon.communicator.RunCmd(node, "kubectl delete -n rook-system daemonset rook-agent")
	addon.communicator.RunCmd(node, "kubectl delete -f https://raw.githubusercontent.com/rook/rook/master/cluster/examples/kubernetes/ceph/operator.yaml")
	addon.communicator.RunCmd(node, "kubectl delete clusterroles rook-agent")
	addon.communicator.RunCmd(node, "kubectl delete clusterrolebindings rook-agent")
	time.Sleep(20 * time.Second)
	addon.communicator.RunCmd(node, "kubectl delete namespace rook")

	for _, node := range addon.nodes {
		if node.IsEtcd || node.IsMaster {
			continue
		}
		fmt.Printf("deleting rook on node %s\n", node.Name)
		addon.communicator.RunCmd(node, "rm -rf /var/lib/rook")
	}

	fmt.Println("Rook uninstalled")
}
