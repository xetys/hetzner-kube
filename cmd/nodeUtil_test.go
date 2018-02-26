package cmd

import (
	"testing"
	"github.com/andreyvit/diff"
)

func TestGenerateMasterConfiguration(t *testing.T) {
	expectedConf := `apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
api:
  advertiseAddress: 10.0.0.1
networking:
  podSubnet: 10.244.0.0/16
apiServerCertSANs:
  - 1.1.1.1
  - 10.0.1.11
  - 127.0.0.1
`

	expectedConfWithEtcd := `apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
api:
  advertiseAddress: 10.0.0.1
networking:
  podSubnet: 10.244.0.0/16
apiServerCertSANs:
  - 1.1.1.1
  - 10.0.1.11
  - 127.0.0.1
etcd:
  endpoints:
  - http://10.0.0.1:2379
  - http://10.0.0.2:2379
`
	nodes := []Node{
		{Name: "node1", IPAddress: "1.1.1.1", PrivateIPAddress: "10.0.0.1", },
		{Name: "node2", IPAddress: "1.1.1.2", PrivateIPAddress: "10.0.0.2", },
	}

	noEtcdConf := GenerateMasterConfiguration(nodes[0], nil)

	if noEtcdConf != expectedConf {
		t.Errorf("master config without etcd does not match to expected.\n%s\n", noEtcdConf)
	}

	etcdConf := GenerateMasterConfiguration(nodes[0], nodes)

	if etcdConf != expectedConfWithEtcd {
		t.Errorf("master config with etcd does not match to expected.\n%s\n", diff.LineDiff(etcdConf, expectedConfWithEtcd))
	}

}
