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
		return fmt.Errorf("cannot peform backup when no etcd nodes are available\n")
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

	if len(etcdNodes) == 0 {
		return false, fmt.Errorf("cannot peform backup when no etcd nodes are available\n")
	}

	firstEtcdNode := etcdNodes[0]

	// check if snapshot exists
	snapshotPath := fmt.Sprintf("/root/etcd-snapshots/%s.db", name)

	_, err := manager.nodeCommunicator.RunCmd(firstEtcdNode, fmt.Sprintf("stat %s", snapshotPath))
	if err != nil {
		return false, fmt.Errorf("cloud not find snapshot '%s' on server", name)
	}

	initialCluster := ""

	// distribute snapshots to all etcd nodes
	fmt.Println("distributing snapshots across all nodes")
	for _, node := range etcdNodes {
		initialCluster += fmt.Sprintf(",%s=http://%s:2380", node.Name, node.PrivateIPAddress)

		if !skipCopy {
			if node.Name == firstEtcdNode.Name {
				continue
			}

			_, err := manager.nodeCommunicator.RunCmd(node, "mkdir -p ~/etcd-snapshots")
			if err != nil {
				return false, err
			}

			err = manager.nodeCommunicator.CopyFileOverNode(firstEtcdNode, node, snapshotPath)
			if err != nil {
				return false, err
			}
			fmt.Printf("copied '%s' to node '%s'\n", snapshotPath, node.Name)
		}
	}

	initialCluster = initialCluster[1:]

	// actual restore the cluster
	fmt.Println("begin restore process")
	for _, node := range etcdNodes {
		// stop etcd
		_, err := manager.nodeCommunicator.RunCmd(node, "systemctl stop etcd.service && rm -rf /var/lib/etcd")
		if err != nil {
			return false, err
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
			return false, err
		}

		_, err = manager.nodeCommunicator.RunCmd(node, "systemctl start etcd.service")
		if err != nil {
			return false, err
		}

		fmt.Printf("etcd node '%s' restored \n", node.Name)
	}

	return true, nil
}

// generateName returns a datetime string for unnamed snapshots
func generateName() string {
	t := time.Now()

	return t.Format("2006-01-02-15-04")
}
