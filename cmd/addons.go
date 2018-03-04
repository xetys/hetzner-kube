package cmd

type ClusterAddon interface {
	Install(args ...string)
	Uninstall()
}

func AddonExists(addonName string) bool {
	switch addonName {
	case "helm", "rook", "ingress", "openebs", "cert-manager":
		return true
	default:
		return false
	}
}

func (cluster Cluster) GetAddon(addonName string) ClusterAddon {
	switch addonName {
	case "helm":
		return NewHelmAddon(cluster)
	case "rook":
		return NewRookAddon(cluster)
	case "ingress":
		return NewIngressAddon(cluster)
	case "openebs":
		return NewOpenEBSAddon(cluster)
	case "cert-manager":
		return NewCertmanagerAddon(cluster)
	default:
		return nil
	}
}
