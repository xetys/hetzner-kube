package hetzner

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

func getDefaultProviderWithNodes() ([]clustermanager.Node, Provider) {
	nodes := []clustermanager.Node{
		{Name: "kube1", IPAddress: "1.1.1.1", PrivateIPAddress: "10.0.1.11", IsEtcd: true},
		{Name: "kube2", IPAddress: "1.1.1.2", PrivateIPAddress: "10.0.1.12", IsMaster: true},
		{Name: "kube3", IPAddress: "1.1.1.3", PrivateIPAddress: "10.0.1.13"},
	}
	provider := Provider{nodes: nodes}
	return nodes, provider
}

func TestCluster_CreateEtcdNodes(t *testing.T) {
	nodes, provider := getDefaultProviderWithNodes()
	etcdNodes := provider.GetEtcdNodes()

	if len(etcdNodes) != 1 {
		t.Error("found more than one etcd node")
	}

	if etcdNodes[0].Name != nodes[0].Name {
		t.Error("wrong node found")
	}
}

func TestProvider_GetMasterNodes(t *testing.T) {
	nodes, provider := getDefaultProviderWithNodes()
	masterNodes := provider.GetMasterNodes()

	if len(masterNodes) != 1 {
		t.Error("found more than one maser node")
	}

	if masterNodes[0].Name != nodes[1].Name {
		t.Error("wrong node found")
	}
}

func TestProvider_CreateWorkerNodes(t *testing.T) {
	nodes, provider := getDefaultProviderWithNodes()
	workerNodes := provider.GetWorkerNodes()

	if len(workerNodes) != 1 {
		t.Error("found more than one worker node")
	}

	if workerNodes[0].Name != nodes[2].Name {
		t.Error("wrong node found")
	}
}

func TestProvider_GetMasterNode(t *testing.T) {
	nodes, provider := getDefaultProviderWithNodes()

	masterNode, err := provider.GetMasterNode()

	if err != nil {
		t.Error(err)
	}

	if masterNode.Name != nodes[1].Name {
		t.Error("master node not found")
	}

	provider.SetNodes([]clustermanager.Node{})

	masterNode, err = provider.GetMasterNode()

	if err == nil {
		t.Error("no error ommited with no master")
	}
}

func TestProvider_GetAllNodes(t *testing.T) {
	nodes, provider := getDefaultProviderWithNodes()
	allNodes := provider.GetAllNodes()
	assert.Equal(t, allNodes, nodes)
}
