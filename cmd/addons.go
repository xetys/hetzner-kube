package cmd

type ClusterAddon interface {
	Install(args ... string)
	Uninstall()
}

func AddonExists(addonName string) bool {
	switch addonName {
	case "helm", "rook", "ingress", "openebs":
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
	default:
		return nil
	}
}
