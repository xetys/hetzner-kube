package clustermanager

// FilePermission is the date uesd to define file permission
type FilePermission string

const (
	// OwnerRead indicate that the file can be readed only from owner
	OwnerRead FilePermission = "C0600"
	// AllRead indicate that the file can be readed from all user on system
	AllRead FilePermission = "C0644"
	// AllExecute indicate that the file can be executed from all user on system
	AllExecute FilePermission = "C0755"
)

// NodeCommunicator is the interface used to define a node comunication protocol
type NodeCommunicator interface {
	RunCmd(node Node, command string) (string, error)
	WriteFile(node Node, filePath string, content string, permission FilePermission) error
	CopyFileOverNode(source Node, target Node, filePath string) error
	TransformFileOverNode(source Node, target Node, filePath string, transform func(string) string) error
}

// EventService is the interface used to manage events
type EventService interface {
	AddEvent(eventName string, eventMessage string)
}

// ClusterProvider is the interface used to declare a cluster provider
type ClusterProvider interface {
	SetNodes([]Node)
	GetAllNodes() []Node
	GetMasterNodes() []Node
	GetEtcdNodes() []Node
	GetWorkerNodes() []Node
	GetMasterNode() (*Node, error)
	GetCluster() Cluster
	GetAdditionalMasterInstallCommands() []NodeCommand
	GetNodeCidr() string
	MustWait() bool
}
