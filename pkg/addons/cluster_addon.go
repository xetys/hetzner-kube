package addons

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

// ClusterAddon describes what functions a cluster addon should provide, so the addon system can use it for the cmd
type ClusterAddon interface {
	Name() string
	Requires() []string
	Description() string
	URL() string
	Install(args ...string)
	Uninstall()
}

// ClusterAddonInitializer is a function creating ClusterAddon instances from given parameters
type ClusterAddonInitializer func(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon

// ClusterAddonService provide the addon service
type ClusterAddonService struct {
	provider         clustermanager.ClusterProvider
	nodeCommunicator clustermanager.NodeCommunicator
	addons           []ClusterAddon
}

// NewClusterAddonService creates an instance of the cluster addon service
func NewClusterAddonService(provider clustermanager.ClusterProvider, nodeComm clustermanager.NodeCommunicator) *ClusterAddonService {
	clusterAddons := []ClusterAddon{
		NewCertmanagerAddon(provider, nodeComm),
		NewDashboardAddon(provider, nodeComm),
		NewDockerregistryAddon(provider, nodeComm),
		NewHCloudControllerManagerAddon(provider, nodeComm),
		NewHelmAddon(provider, nodeComm),
		NewHetznerCSIAddon(provider, nodeComm),
		NewIngressAddon(provider, nodeComm),
		NewOpenEBSAddon(provider, nodeComm),
		NewPrometheusAddon(provider, nodeComm),
		NewRookAddon(provider, nodeComm),
	}

	return &ClusterAddonService{
		provider:         provider,
		nodeCommunicator: nodeComm,
		addons:           clusterAddons,
	}
}

// AddonExists return true, if an addon with the requested name exists
func (addonService *ClusterAddonService) AddonExists(addonName string) bool {
	for _, addon := range addonService.addons {
		if addon.Name() == addonName {
			return true
		}
	}

	return false
}

// GetAddon returns the ClusterAddon instance given by name, or nil if not found
func (addonService *ClusterAddonService) GetAddon(addonName string) ClusterAddon {
	for _, addon := range addonService.addons {
		if addon.Name() == addonName {
			return addon
		}
	}

	return nil
}

// Addons returns a list of all addons
func (addonService *ClusterAddonService) Addons() []ClusterAddon {
	return addonService.addons
}
