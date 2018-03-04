package cmd

import "log"

type CertmanagerAddon struct {
	masterNode *Node
}

func NewCertmanagerAddon(cluster Cluster) ClusterAddon {
	masterNode, err := cluster.GetMasterNode()
	FatalOnError(err)
	return CertmanagerAddon{masterNode: masterNode}
}

func (addon CertmanagerAddon) Install(args ... string) {
	node := *addon.masterNode
	_, err := runCmd(node, "helm install --name cert-manager --namespace kube-system stable/cert-manager")
	FatalOnError(err)
	log.Println("cert-manager installed")
}

func (addon CertmanagerAddon) Uninstall() {
	node := *addon.masterNode
	_, err := runCmd(node, "helm delete --purge cert-manager")
	FatalOnError(err)
	log.Println("cert-manager uninstalled")
}
