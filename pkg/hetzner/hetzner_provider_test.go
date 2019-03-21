package hetzner_test

import (
	"context"
	"testing"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/magiconair/properties/assert"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

func getProviderWithNodes(nodes []clustermanager.Node) hetzner.Provider {
	provider := hetzner.Provider{}

	provider.SetNodes(nodes)

	return provider
}

type testCase struct {
	Name         string
	Nodes        []clustermanager.Node
	MatchedNodes []string
}

func getNodeNames(nodes []clustermanager.Node) []string {
	nodeNames := []string{}

	for _, node := range nodes {
		nodeNames = append(nodeNames, node.Name)
	}

	return nodeNames
}

func TestProviderGetMasterNodes(t *testing.T) {
	tests := []testCase{
		{
			Name: "Single master node",
			Nodes: []clustermanager.Node{
				{Name: "kube-master-1", IsMaster: true},
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-worker-1"},
			},
			MatchedNodes: []string{
				"kube-master-1",
			},
		},
		{
			Name: "Two master nodes",
			Nodes: []clustermanager.Node{
				{Name: "kube-master-1", IsMaster: true},
				{Name: "kube-master-2", IsMaster: true},
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-worker-1"},
			},
			MatchedNodes: []string{
				"kube-master-1",
				"kube-master-2",
			},
		},
		{
			Name: "Two etcd node that are also master",
			Nodes: []clustermanager.Node{
				{Name: "kube-etcd-1", IsMaster: true, IsEtcd: true},
				{Name: "kube-etcd-2", IsMaster: true, IsEtcd: true},
				{Name: "kube-etcd-3", IsEtcd: true},
				{Name: "kube-worker-1"},
			},
			MatchedNodes: []string{
				"kube-etcd-1",
				"kube-etcd-2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			provider := getProviderWithNodes(tt.Nodes)
			nodes := provider.GetMasterNodes()

			assert.Equal(t, getNodeNames(nodes), tt.MatchedNodes)
		})
	}
}

func TestProviderGetEtcdNodes(t *testing.T) {
	tests := []testCase{
		{
			Name: "Single etcd node",
			Nodes: []clustermanager.Node{
				{Name: "kube-master-1", IsMaster: true},
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-worker-1"},
			},
			MatchedNodes: []string{
				"kube-etcd-1",
			},
		},
		{
			Name: "Two etcd nodes",
			Nodes: []clustermanager.Node{
				{Name: "kube-master-1", IsMaster: true},
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-etcd-2", IsEtcd: true},
				{Name: "kube-worker-1"},
			},
			MatchedNodes: []string{
				"kube-etcd-1",
				"kube-etcd-2",
			},
		},
		{
			Name: "Three etcd node some of them are also master",
			Nodes: []clustermanager.Node{
				{Name: "kube-etcd-1", IsMaster: true, IsEtcd: true},
				{Name: "kube-etcd-2", IsMaster: true, IsEtcd: true},
				{Name: "kube-etcd-3", IsEtcd: true},
				{Name: "kube-worker-1"},
			},
			MatchedNodes: []string{
				"kube-etcd-1",
				"kube-etcd-2",
				"kube-etcd-3",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			provider := getProviderWithNodes(tt.Nodes)
			nodes := provider.GetEtcdNodes()

			assert.Equal(t, getNodeNames(nodes), tt.MatchedNodes)
		})
	}
}

func TestProviderGetWorkerNodes(t *testing.T) {
	tests := []testCase{
		{
			Name: "Single worker node",
			Nodes: []clustermanager.Node{
				{Name: "kube-master-1", IsMaster: true},
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-worker-1"},
			},
			MatchedNodes: []string{
				"kube-worker-1",
			},
		},
		{
			Name: "Two worker nodes",
			Nodes: []clustermanager.Node{
				{Name: "kube-master-1", IsMaster: true},
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-worker-1"},
				{Name: "kube-worker-2"},
			},
			MatchedNodes: []string{
				"kube-worker-1",
				"kube-worker-2",
			},
		},
		{
			Name: "No worker nodes",
			Nodes: []clustermanager.Node{
				{Name: "kube-etcd-1", IsMaster: true, IsEtcd: true},
				{Name: "kube-etcd-2", IsMaster: true, IsEtcd: true},
				{Name: "kube-etcd-3", IsEtcd: true},
			},
			MatchedNodes: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			provider := getProviderWithNodes(tt.Nodes)
			nodes := provider.GetWorkerNodes()

			assert.Equal(t, getNodeNames(nodes), tt.MatchedNodes)
		})
	}
}

func TestProviderGetAllNodes(t *testing.T) {
	tests := []testCase{
		{
			Name: "One node per type",
			Nodes: []clustermanager.Node{
				{Name: "kube-master-1", IsMaster: true},
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-worker-1"},
			},
			MatchedNodes: []string{
				"kube-master-1",
				"kube-etcd-1",
				"kube-worker-1",
			},
		},
		{
			Name: "Multiple node per type",
			Nodes: []clustermanager.Node{
				{Name: "kube-master-1", IsMaster: true},
				{Name: "kube-master-2", IsMaster: true},
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-etcd-2", IsEtcd: true},
				{Name: "kube-worker-1"},
				{Name: "kube-worker-2"},
				{Name: "kube-worker-3"},
			},
			MatchedNodes: []string{
				"kube-master-1",
				"kube-master-2",
				"kube-etcd-1",
				"kube-etcd-2",
				"kube-worker-1",
				"kube-worker-2",
				"kube-worker-3",
			},
		},
		{
			Name:         "No nodes",
			Nodes:        []clustermanager.Node{},
			MatchedNodes: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			provider := getProviderWithNodes(tt.Nodes)
			nodes := provider.GetAllNodes()

			assert.Equal(t, getNodeNames(nodes), tt.MatchedNodes)
		})
	}
}

func TestProviderGetMasterNode(t *testing.T) {
	tests := []testCase{
		{
			Name: "Single master node",
			Nodes: []clustermanager.Node{
				{Name: "kube-master-1", IsMaster: true},
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-worker-1"},
			},
			MatchedNodes: []string{"kube-master-1"},
		},
		{
			Name: "Two master nodes",
			Nodes: []clustermanager.Node{
				{Name: "kube-master-1", IsMaster: true},
				{Name: "kube-master-2", IsMaster: true},
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-worker-1"},
			},
			MatchedNodes: []string{"kube-master-1"},
		},
		{
			Name: "An etcd node that is also master",
			Nodes: []clustermanager.Node{
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-etcd-2", IsMaster: true, IsEtcd: true},
				{Name: "kube-etcd-3", IsEtcd: true},
				{Name: "kube-master-1", IsMaster: true},
				{Name: "kube-worker-1"},
			},
			MatchedNodes: []string{"kube-etcd-2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			provider := getProviderWithNodes(tt.Nodes)
			node, _ := provider.GetMasterNode()

			assert.Equal(t, []string{node.Name}, tt.MatchedNodes)
		})
	}
}

func TestProviderGetMasterNodeIsMissing(t *testing.T) {
	tests := []struct {
		Name  string
		Nodes []clustermanager.Node
	}{
		{
			Name:  "No nodes",
			Nodes: []clustermanager.Node{},
		},
		{
			Name: "No master nodes",
			Nodes: []clustermanager.Node{
				{Name: "kube-etcd-1", IsEtcd: true},
				{Name: "kube-worker-1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			provider := getProviderWithNodes(tt.Nodes)
			_, err := provider.GetMasterNode()

			if err == nil {
				t.Error("no error omitted with no master")
			}
		})
	}
}

func TestProviderInitCluster(t *testing.T) {
	provider := getProviderWithNodes([]clustermanager.Node{})

	_, err := provider.GetMasterNode()

	if err == nil {
		t.Error("no error omitted with no master")
	}
}

func TestProviderGetCluster(t *testing.T) {
	nodes := []clustermanager.Node{
		{Name: "kube-etcd-1", IsEtcd: true},
		{Name: "kube-etcd-2", IsMaster: true, IsEtcd: true},
		{Name: "kube-etcd-3", IsEtcd: true},
		{Name: "kube-master-1", IsMaster: true},
		{Name: "kube-worker-1"},
	}

	provider := hetzner.NewHetznerProvider(
		context.Background(),
		&hcloud.Client{},
		clustermanager.Cluster{
			Name:          "cluster-name",
			NodeCIDR:      "10.0.1.0/24",
			CloudInitFile: "cloud/init.file",
		},
		"token-string",
	)

	provider.SetNodes(nodes)

	cluster := provider.GetCluster()
	expectedCluster := clustermanager.Cluster{
		Name:          "cluster-name",
		NodeCIDR:      "10.0.1.0/24",
		HaEnabled:     false,
		IsolatedEtcd:  false,
		CloudInitFile: "cloud/init.file",
		Nodes:         nodes,
	}

	assert.Equal(t, cluster, expectedCluster)
}
