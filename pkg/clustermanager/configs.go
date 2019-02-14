package clustermanager

import (
	"fmt"
	"strings"
)

// GenerateMasterConfiguration generate the kubernetes config for master
func GenerateMasterConfiguration(masterNode Node, masterNodes, etcdNodes []Node) string {
	masterConfigTpl := `apiVersion: kubeadm.k8s.io/v1alpha3
kind: ClusterConfiguration
networking:
  serviceSubnet: "10.96.0.0/12"
  podSubnet: "10.244.0.0/16"
  dnsDomain: "cluster.local"
apiServerCertSANs:
  - 127.0.0.1
  - %s
%s%s
---
apiVersion: kubeadm.k8s.io/v1alpha3
kind: InitConfiguration
apiEndpoint:
  advertiseAddress: %s
  bindPort: 6443
nodeRegistration:
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
`

	masterNodesIps := ""
	for _, node := range masterNodes {
		masterNodesIps = fmt.Sprintf("%s  - %s\n", masterNodesIps, node.PrivateIPAddress)
	}

	etcdConfig := ""
	if len(etcdNodes) > 0 {
		etcdConfig = `etcd:
  external:
    endpoints:` + "\n"

		for _, node := range etcdNodes {
			etcdConfig = fmt.Sprintf("%s%s\n", etcdConfig, "    - http://"+node.PrivateIPAddress+":2379")
		}
	}

	masterConfig := fmt.Sprintf(
		masterConfigTpl,
		masterNode.IPAddress,
		masterNodesIps,
		etcdConfig,
		masterNode.PrivateIPAddress,
	)

	return masterConfig
}

// GenerateEtcdSystemdService generate configuration file used to manage etcd service on systemd
func GenerateEtcdSystemdService(node Node, etcdNodes []Node) string {
	serviceTpls := `# /etc/systemd/system/etcd.service
[Unit]
Description=etcd
After=network.target wg-quick@wg0.service

[Service]
ExecStart=/opt/etcd/etcd --name %s \
  --data-dir /var/lib/etcd \
  --listen-client-urls "http://%s:2379,http://localhost:2379" \
  --advertise-client-urls "http://%s:2379" \
  --listen-peer-urls "http://%s:2380" \
  --initial-cluster "%s" \
  --initial-advertise-peer-urls "http://%s:2380" \
  --heartbeat-interval 200 \
  --election-timeout 5000
Restart=always
RestartSec=5
TimeoutStartSec=0
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
`

	ips := make([]string, len(etcdNodes))
	for i, node := range etcdNodes {
		ips[i] = fmt.Sprintf("%s=http://%s:2380", node.Name, node.PrivateIPAddress)
	}
	initialCluster := strings.Join(ips, ",")

	service := fmt.Sprintf(
		serviceTpls,
		node.Name,
		node.PrivateIPAddress,
		node.PrivateIPAddress,
		node.PrivateIPAddress,
		initialCluster,
		node.PrivateIPAddress,
	)

	return service
}
