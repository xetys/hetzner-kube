package cmd

import "log"

type DockerregistryAddon struct {
	masterNode *Node
}

func NewDockerregistryAddon(cluster Cluster) ClusterAddon {
	masterNode, err := cluster.GetMasterNode()
	FatalOnError(err)
	return DockerregistryAddon{masterNode: masterNode}
}

func (addon DockerregistryAddon) Install(args ... string) {
	node := *addon.masterNode
	_, err := runCmd(node, "helm install --set persistence.enabled=true stable/docker-registry")
	FatalOnError(err)
	log.Println("docker-registry installed")
}

func (addon DockerregistryAddon) Uninstall() {
	node := *addon.masterNode
	_, err := runCmd(node, "helm delete --purge docker-registry")
	FatalOnError(err)
	log.Println("docker-registry uninstalled")
}
