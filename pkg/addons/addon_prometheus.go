package addons

import (
	"fmt"

	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

// PrometheusAddon provides cluster monitoring using prometheus operator
type PrometheusAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
	nodes        []clustermanager.Node
	provider     *hetzner.Provider
}

// NewPrometheusAddon create a new prometheus addon
func NewPrometheusAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, err := provider.GetMasterNode()
	FatalOnError(err)
	return &PrometheusAddon{
		masterNode:   masterNode,
		communicator: communicator,
		nodes:        provider.GetAllNodes(),
		provider:     provider.(*hetzner.Provider),
	}
}

func init() {
	addAddon(NewPrometheusAddon)
}

// Name returns the addons name
func (addon *PrometheusAddon) Name() string {
	return "kube-prometheus"
}

// Requires returns a slice with the name of required addons
func (addon *PrometheusAddon) Requires() []string {
	return []string{}
}

// Description returns the addons description
func (addon *PrometheusAddon) Description() string {
	return "CoreOS prometheus operator /w cluster monitoring"
}

// URL returns the URL of the addons underlying project
func (addon *PrometheusAddon) URL() string {
	return "https://github.com/coreos/prometheus-operator"
}

// Install installs the prometheus operator
func (addon *PrometheusAddon) Install(args ...string) {
	// apply cAdvisor and kubelet config
	kubeletModifyScript := `#!/bin/bash

KUBEADM_SYSTEMD_CONF=/etc/systemd/system/kubelet.service.d/10-kubeadm.conf
sed -e "/cadvisor-port=0/d" -i "$KUBEADM_SYSTEMD_CONF"
if ! grep -q "authentication-token-webhook=true" "$KUBEADM_SYSTEMD_CONF"; then
  sed -e "s/--authorization-mode=Webhook/--authentication-token-webhook=true --authorization-mode=Webhook/" -i "$KUBEADM_SYSTEMD_CONF"
fi
systemctl daemon-reload
systemctl restart kubelet`

	for _, node := range addon.nodes {
		err := addon.communicator.WriteFile(node, "/tmp/prometheus.sh", kubeletModifyScript, clustermanager.AllExecute)
		FatalOnError(err)
		_, err = addon.communicator.RunCmd(node, "/tmp/prometheus.sh")
		FatalOnError(err)

		if node.IsMaster {
			_, err = addon.communicator.RunCmd(node, `sed -e "s/- --address=127.0.0.1/- --address=0.0.0.0/" -i /etc/kubernetes/manifests/kube-controller-manager.yaml`)
			FatalOnError(err)
			_, err = addon.communicator.RunCmd(node, `sed -e "s/- --address=127.0.0.1/- --address=0.0.0.0/" -i /etc/kubernetes/manifests/kube-scheduler.yaml`)
			FatalOnError(err)
		}
	}

	// get the repo
	addon.run("git clone --branch release-0.19 --depth 1 https://github.com/coreos/prometheus-operator")
	// get the customized manifests
	addon.run("cd /root/prometheus-operator/contrib/kube-prometheus/example-dist && git clone https://github.com/xetys/hetzner-kube-prometheus")
	// install the operator
	addon.run("cd /root/prometheus-operator/contrib/kube-prometheus && ./hack/cluster-monitoring/deploy example-dist/hetzner-kube-prometheus")
}

// Uninstall removes the prometheus operator
func (addon *PrometheusAddon) Uninstall() {
	addon.run("cd /root/prometheus-operator/contrib/kube-prometheus/ && ./hack/cluster-monitoring/teardown")
	addon.run("rm -rf prometheus-operator")
}

func (addon *PrometheusAddon) run(cmd string) {
	out, err := addon.communicator.RunCmd(*addon.masterNode, cmd)
	if err != nil {
		fmt.Printf("Failing command:\n%s\nOutput: %s\n\n", cmd, out)
		FatalOnError(err)
	}
}
