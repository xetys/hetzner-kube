package clustermanager

//NodeCommunicator is the interface used to define a node comunication protocol
type NodeCommunicator interface {
	RunCmd(node Node, command string) (string, error)
	WriteFile(node Node, filePath string, content string, executable bool) error
	CopyFileOverNode(source Node, target Node, filePath string) error
	TransformFileOverNode(source Node, target Node, filePath string, transform func(string) string) error
}

//EventService is the interface used to manage events
type EventService interface {
	AddEvent(eventName string, eventMessage string)
}

//ClusterProvider is the interface used to declare a cluster provider
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
