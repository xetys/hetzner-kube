package clustermanager

type NodeCommunicator interface {
	RunCmd(node Node, command string) (string, error)
	WriteFile(node Node, filePath string, content string, executable bool) error
	CopyFileOverNode(source Node, target Node, filePath string) error
	TransformFileOverNode(source Node, target Node, filePath string, transform func(string) string) error
}

type EventService interface {
	AddEvent(eventName string, eventMessage string)
}

type ClusterProvider interface {
	GetAllNodes() []Node
	GetMasterNodes() []Node
	GetEtcdNodes() []Node
	GetWorkerNodes() []Node
	GetMasterNode() (*Node, error)
	GetCluster() Cluster
	GetAdditionalMasterInstallCommands() []NodeCommand
}
