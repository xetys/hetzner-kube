package clustermanager

import (
	"fmt"
	"strings"
)

func GenerateMasterConfiguration(masterNode Node, masterNodes, etcdNodes []Node) string {
	masterConfigTpl := `apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
api:
  advertiseAddress: %s
networking:
  podSubnet: 10.244.0.0/16
apiServerCertSANs:
  - %s
  - 127.0.0.1
`
	etcdConfig := `etcd:
  endpoints:`
	masterConfig := fmt.Sprintf(masterConfigTpl, masterNode.PrivateIPAddress, masterNode.IPAddress)
	for _, node := range masterNodes {
		masterConfig = fmt.Sprintf("%s%s\n", masterConfig, "  - "+node.PrivateIPAddress)
	}

	if len(etcdNodes) > 0 {
		masterConfig = masterConfig + etcdConfig + "\n"
		for _, node := range etcdNodes {
			masterConfig = fmt.Sprintf("%s%s\n", masterConfig, "  - http://"+node.PrivateIPAddress+":2379")
		}
	}

	return masterConfig
}

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

	var ips []string
	for _, node := range etcdNodes {
		ips = append(ips, fmt.Sprintf("%s=http://%s:2380", node.Name, node.PrivateIPAddress))
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
