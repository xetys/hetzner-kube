package clustermanager

import (
	"testing"

	"github.com/andreyvit/diff"
)

func TestGenerateMasterConfiguration(t *testing.T) {
	expectedConf := `apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
kubernetesVersion: stable
networking:
  serviceSubnet: "10.96.0.0/12"
  podSubnet: "10.244.0.0/16"
  dnsDomain: "cluster.local"
apiServer:
  featureGates:
    CSINodeInfo: true
    CSIDriverRegistry: true
  certSANs:
    - 127.0.0.1
    - 1.1.1.1
    - 10.0.0.1
    - 10.0.0.2

---
apiVersion: kubeadm.k8s.io/v1beta1
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: 10.0.0.1
  bindPort: 6443
nodeRegistration:
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
featureGates:
  CSINodeInfo: true
  CSIDriverRegistry: true
`

	expectedConfWithEtcd := `apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
kubernetesVersion: stable
networking:
  serviceSubnet: "10.96.0.0/12"
  podSubnet: "10.244.0.0/16"
  dnsDomain: "cluster.local"
apiServer:
  featureGates:
    CSINodeInfo: true
    CSIDriverRegistry: true
  certSANs:
    - 127.0.0.1
    - 1.1.1.1
    - 10.0.0.1
    - 10.0.0.2
etcd:
  external:
    endpoints:
    - http://10.0.0.1:2379
    - http://10.0.0.2:2379

---
apiVersion: kubeadm.k8s.io/v1beta1
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: 10.0.0.1
  bindPort: 6443
nodeRegistration:
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
featureGates:
  CSINodeInfo: true
  CSIDriverRegistry: true
`
	nodes := []Node{
		{Name: "node1", IPAddress: "1.1.1.1", PrivateIPAddress: "10.0.0.1"},
		{Name: "node2", IPAddress: "1.1.1.2", PrivateIPAddress: "10.0.0.2"},
	}

	noEtcdConf := GenerateMasterConfiguration(nodes[0], nodes, nil)

	if noEtcdConf != expectedConf {
		t.Errorf("master config without etcd does not match to expected.\n%s\n", diff.LineDiff(noEtcdConf, expectedConf))
	}

	etcdConf := GenerateMasterConfiguration(nodes[0], nodes, nodes)

	if etcdConf != expectedConfWithEtcd {
		t.Errorf("master config with etcd does not match to expected.\n%s\n", diff.LineDiff(etcdConf, expectedConfWithEtcd))
	}
}

func TestGenerateEtcdSystemdService(t *testing.T) {
	expectedString := `# /etc/systemd/system/etcd.service
[Unit]
Description=etcd
After=network.target wg-quick@wg0.service

[Service]
ExecStart=/opt/etcd/etcd --name kube1 \
  --data-dir /var/lib/etcd \
  --listen-client-urls "http://10.0.1.11:2379,http://localhost:2379" \
  --advertise-client-urls "http://10.0.1.11:2379" \
  --listen-peer-urls "http://10.0.1.11:2380" \
  --initial-cluster "kube1=http://10.0.1.11:2380,kube2=http://10.0.1.12:2380,kube3=http://10.0.1.13:2380" \
  --initial-advertise-peer-urls "http://10.0.1.11:2380" \
  --heartbeat-interval 200 \
  --election-timeout 5000
Restart=always
RestartSec=5
TimeoutStartSec=0
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
`
	nodes := []Node{
		{Name: "kube1", IPAddress: "1.1.1.1", PrivateIPAddress: "10.0.1.11"},
		{Name: "kube2", IPAddress: "1.1.1.2", PrivateIPAddress: "10.0.1.12"},
		{Name: "kube3", IPAddress: "1.1.1.3", PrivateIPAddress: "10.0.1.13"},
	}

	etcdService := GenerateEtcdSystemdService(nodes[0], nodes)

	if etcdService != expectedString {
		t.Errorf("etcd systemd service does not match expected\n%s", diff.LineDiff(expectedString, etcdService))
	}
}
