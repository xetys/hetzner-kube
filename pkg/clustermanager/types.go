package clustermanager

//Node is the structure used to define a node
type Node struct {
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Group            string    `json:"group"`
	IsMaster         bool      `json:"is_master"`
	IsEtcd           bool      `json:"is_etcd"`
	IPAddress        string    `json:"ip_address"`
	PrivateIPAddress string    `json:"private_ip_address"`
	SSHKeyName       string    `json:"ssh_key_name"`
	WireGuardKeyPair WgKeyPair `json:"wire_guard_key_pair"`
}

//Cluster is the structure used to define a cluster
type Cluster struct {
	Name          string `json:"name"`
	Nodes         []Node `json:"nodes"`
	HaEnabled     bool   `json:"ha_enabled"`
	IsolatedEtcd  bool   `json:"isolated_etcd"`
	SelfHosted    bool   `json:"self_hosted"`
	CloudInitFile string `json:"cloud_init_file"`
	NodeCIDR      string `json:"node_cidr"`
}

//NodeCommand is the structure used to define acommand to execute on a node
type NodeCommand struct {
	EventName string
	Command   string
}
