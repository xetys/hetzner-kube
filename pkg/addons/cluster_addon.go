package addons

import "github.com/xetys/hetzner-kube/pkg/clustermanager"

type ClusterAddon interface {
	Install(args ...string)
	Uninstall()
}

type ClusterAddonService struct {
	provider clustermanager.ClusterProvider
	nodeCommunicator clustermanager.NodeCommunicator
}

func NewClusterAddonService(provider clustermanager.ClusterProvider, nodeComm clustermanager.NodeCommunicator) *ClusterAddonService {
	return &ClusterAddonService{provider: provider, nodeCommunicator: nodeComm}
}

func (ClusterAddonService) AddonExists(addonName string) bool {
	switch addonName {
	case "helm", "rook", "ingress", "openebs", "cert-manager", "docker-registry":
		return true
	default:
		return false
	}
}

func (addonService ClusterAddonService) GetAddon(addonName string) ClusterAddon {
	switch addonName {
	case "helm":
		return NewHelmAddon(addonService.provider, addonService.nodeCommunicator)
	case "rook":
		return NewRookAddon(addonService.provider, addonService.nodeCommunicator)
	case "ingress":
		return NewIngressAddon(addonService.provider, addonService.nodeCommunicator)
	case "openebs":
		return NewOpenEBSAddon(addonService.provider, addonService.nodeCommunicator)
	case "cert-manager":
		return NewCertmanagerAddon(addonService.provider, addonService.nodeCommunicator)
	case "docker-registry":
		return NewDockerregistryAddon(addonService.provider, addonService.nodeCommunicator)
	default:
		return nil
	}
}
