package addons

import (
	"fmt"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

type HelmAddon struct {
	masterNode *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

func NewHelmAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return HelmAddon{masterNode: masterNode, communicator: communicator}
}

func (addon HelmAddon) Install(args ...string) {

	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get | bash")
	FatalOnError(err)
	serviceAccount := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: tiller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system`
	err = addon.communicator.WriteFile(node, "/root/helm-service-account.yaml", serviceAccount, false)
	FatalOnError(err)

	_, err = addon.communicator.RunCmd(node, "kubectl apply -f helm-service-account.yaml")
	FatalOnError(err)

	_, err = addon.communicator.RunCmd(node, "helm init --service-account tiller")
	FatalOnError(err)

	fmt.Println("Helm installed")
}

func (addon HelmAddon) Uninstall() {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "helm reset --force")
	FatalOnError(err)

	fmt.Println("Helm uninstalled")
}
