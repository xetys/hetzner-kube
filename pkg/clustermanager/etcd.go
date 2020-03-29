package clustermanager

import (
	"fmt"
	"time"
)

// EtcdManager is a tool which provides basic backup & restore functionality for HA clusters
type EtcdManager struct {
	provider         ClusterProvider
	nodeCommunicator NodeCommunicator
}

// NewEtcdManager returns a new instance of EtcdManager
func NewEtcdManager(provider ClusterProvider, nodeCommunicator NodeCommunicator) *EtcdManager {
	return &EtcdManager{
		provider:         provider,
		nodeCommunicator: nodeCommunicator,
	}
}

// CreateSnapshot creates a snapshot with a name. If name is empty, a datetime string is generated
func (manager *EtcdManager) CreateSnapshot(name string) error {
	// create snapshot
	etcdNodes := manager.provider.GetEtcdNodes()

	if len(etcdNodes) == 0 {
		return fmt.Errorf("cannot peform backup when no etcd nodes are available")
	}

	firstEtcdNode := etcdNodes[0]
	snapshotName := name

	if snapshotName == "" {
		snapshotName = generateName()
	}

	_, err := manager.nodeCommunicator.RunCmd(firstEtcdNode, "mkdir -p ~/etcd-snapshots")
	if err != nil {
		return err
	}

	saveCommand := fmt.Sprintf("ETCDCTL_API=3 /opt/etcd/etcdctl snapshot save ~/etcd-snapshots/%s.db", snapshotName)
	_, err = manager.nodeCommunicator.RunCmd(firstEtcdNode, saveCommand)

	if err != nil {
		return err
	}

	return nil
}

// RestoreSnapshot restores a snapshot, given its name
func (manager *EtcdManager) RestoreSnapshot(name string, skipCopy bool) (bool, error) {
	etcdNodes := manager.provider.GetEtcdNodes()
	snapshotPath := fmt.Sprintf("/root/etcd-snapshots/%s.db", name)

	if len(etcdNodes) == 0 {
		return false, fmt.Errorf("cannot peform backup when no etcd nodes are available")
	}

	firstEtcdNode := etcdNodes[0]

	// check if snapshot exists
	_, err := manager.nodeCommunicator.RunCmd(firstEtcdNode, fmt.Sprintf("stat %s", snapshotPath))
	if err != nil {
		return false, fmt.Errorf("cloud not find snapshot '%s' on server", name)
	}

	err = manager.copyAndRestore(firstEtcdNode, snapshotPath, skipCopy)

	return err == nil, err
}

// copySnapshot copies a snapshot to a node
func (manager *EtcdManager) copySnapshot(firstEtcdNode, node Node, snapshotPath string) error {
	_, err := manager.nodeCommunicator.RunCmd(node, "mkdir -p ~/etcd-snapshots")
	if err != nil {
		return err
	}

	err = manager.nodeCommunicator.CopyFileOverNode(firstEtcdNode, node, snapshotPath)
	if err != nil {
		return err
	}

	fmt.Printf("copied '%s' to node '%s'\n", snapshotPath, node.Name)

	return nil
}

// restoreNode performs the snapshot restore step1
func (manager *EtcdManager) restoreNode(node Node, snapshotPath, initialCluster string) error {
	// stop etcd
	_, err := manager.nodeCommunicator.RunCmd(node, "systemctl stop etcd.service && rm -rf /var/lib/etcd")
	if err != nil {
		return err
	}

	restoreCmd := fmt.Sprintf("ETCDCTL_API=3 /opt/etcd/etcdctl snapshot restore %s --name %s --data-dir /var/lib/etcd --initial-cluster %s --initial-advertise-peer-urls \"http://%s:2380\"",
		snapshotPath,
		node.Name,
		initialCluster,
		node.PrivateIPAddress,
	)

	out, err := manager.nodeCommunicator.RunCmd(node, restoreCmd)
	if err != nil {
		fmt.Println(out)
		return err
	}

	_, err = manager.nodeCommunicator.RunCmd(node, "systemctl start etcd.service")
	if err != nil {
		return err
	}

	fmt.Printf("etcd node '%s' restored \n", node.Name)

	return nil
}

// copyAndRestore performs the actual tasks for a restore progress1
func (manager *EtcdManager) copyAndRestore(firstEtcdNode Node, snapshotPath string, skipCopy bool) error {
	etcdNodes := manager.provider.GetEtcdNodes()
	initialCluster := ""

	// distribute snapshots to all etcd nodes
	fmt.Println("distributing snapshots across all nodes")

	for _, node := range etcdNodes {
		initialCluster += fmt.Sprintf(",%s=http://%s:2380", node.Name, node.PrivateIPAddress)

		if !skipCopy {
			if node.Name == firstEtcdNode.Name {
				continue
			}

			err := manager.copySnapshot(firstEtcdNode, node, snapshotPath)
			if err != nil {
				return err
			}
		}
	}

	initialCluster = initialCluster[1:]

	// actual restore the cluster
	fmt.Println("begin restore process")

	for _, node := range etcdNodes {
		err := manager.restoreNode(node, snapshotPath, initialCluster)
		if err != nil {
			return err
		}
	}

	return nil
}

// generateName returns a datetime string for unnamed snapshots
func generateName() string {
	t := time.Now()

	return t.Format("2006-01-02-15-04")
}
