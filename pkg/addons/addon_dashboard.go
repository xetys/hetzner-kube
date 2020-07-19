package addons

import (
	"fmt"

	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

// DashboardAddon installs dashboard
type DashboardAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

// NewDashboardAddon installs dashboard to the cluster
func NewDashboardAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return DashboardAddon{masterNode: masterNode, communicator: communicator}
}

func init() {
	addAddon(NewDashboardAddon)
}

// Name returns the addons name
func (addon DashboardAddon) Name() string {
	return "dashboard"
}

// Requires returns a slice with the name of required addons
func (addon DashboardAddon) Requires() []string {
	return []string{}
}

// Description returns the addons description
func (addon DashboardAddon) Description() string {
	return "Kubernetes Dashboard"
}

// URL returns the URL of the addons underlying project
func (addon DashboardAddon) URL() string {
	return "https://github.com/kubernetes/dashboard"
}

// Install performs all steps to install the addon
func (addon DashboardAddon) Install(args ...string) {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.0.3/aio/deploy/recommended.yaml")
	FatalOnError(err)

	serviceAccount := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: admin-user
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: admin-user
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: admin-user
    namespace: kube-system`
	err = addon.communicator.WriteFile(node, "/root/dashboard-service-account.yaml", serviceAccount, clustermanager.OwnerRead)
	FatalOnError(err)

	_, err = addon.communicator.RunCmd(node, "kubectl apply -f dashboard-service-account.yaml")
	FatalOnError(err)

	token, err := addon.communicator.RunCmd(node, "kubectl -n kube-system describe secret $(kubectl -n kube-system get secret | grep admin-user | awk '{print $1}') | grep -E '^token' | cut -f2 -d':' | tr -d ' \t'")
	FatalOnError(err)

	fmt.Printf("Use the following token to login to the dashboard: %s\n", token)
	fmt.Println("Dashboard installed")
}

// Uninstall performs all steps to remove the addon
func (addon DashboardAddon) Uninstall() {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "kubectl delete -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.0.3/aio/deploy/recommended.yaml")
	FatalOnError(err)

	_, err = addon.communicator.RunCmd(node, "kubectl delete -f dashboard-service-account.yaml")
	FatalOnError(err)

	fmt.Println("Dashboard uninstalled")
}
