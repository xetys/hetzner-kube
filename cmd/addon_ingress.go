package cmd

import "log"

type IngressAddon struct {
	masterNode *Node
}

func NewIngressAddon(cluster Cluster) ClusterAddon {
	masterNode, err := cluster.GetMasterNode()
	FatalOnError(err)
	return IngressAddon{masterNode: masterNode}
}

func (addon IngressAddon) Install(args ...string) {
	node := *addon.masterNode
	_, err := runCmd(node, "helm install --name ingress --set rbac.create=true,controller.kind=DaemonSet,controller.service.type=ClusterIP,controller.hostNetwork=true stable/nginx-ingress")
	FatalOnError(err)
	log.Println("nginx ingress installed")
}

func (addon IngressAddon) Uninstall() {
	node := *addon.masterNode
	_, err := runCmd(node, "helm delete --purge ingress")
	FatalOnError(err)
	log.Println("nginx ingress uninstalled")
}
