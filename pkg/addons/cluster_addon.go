package addons

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

type ClusterAddon interface {
	Name() string
	Description() string
	URL() string
	Install(args ...string)
	Uninstall()
}

type ClusterAddonInitializer func(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon

var addonInitializers []ClusterAddonInitializer = make([]ClusterAddonInitializer, 0)

func addAddon(clusterAddon ClusterAddonInitializer) {
	addonInitializers = append(addonInitializers, clusterAddon)
}

type ClusterAddonService struct {
	provider         clustermanager.ClusterProvider
	nodeCommunicator clustermanager.NodeCommunicator
	addons           []ClusterAddon
}

func NewClusterAddonService(provider clustermanager.ClusterProvider, nodeComm clustermanager.NodeCommunicator) *ClusterAddonService {
	clusterAddons := []ClusterAddon{}
	for _, initializer := range addonInitializers {
		clusterAddons = append(clusterAddons, initializer(provider, nodeComm))
	}
	return &ClusterAddonService{provider: provider, nodeCommunicator: nodeComm, addons: clusterAddons}
}

func (addonService *ClusterAddonService) AddonExists(addonName string) bool {
	for _, addon := range addonService.addons {
		if addon.Name() == addonName {
			return true
		}
	}
	return false
}

func (addonService *ClusterAddonService) GetAddon(addonName string) ClusterAddon {
	for _, addon := range addonService.addons {
		if addon.Name() == addonName {
			return addon
		}
	}

	return nil
}

func (addonService *ClusterAddonService) Addons() []ClusterAddon {
	return addonService.addons
}
