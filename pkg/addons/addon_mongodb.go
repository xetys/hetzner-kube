package addons

import (
	"fmt"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

//MongoDbAddon installs MongoDb as ReplicaSet
type MongoDbAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

//NewMongoDbAddon creates an addon which installs MongoDb
func NewMongoDbAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, _ := provider.GetMasterNode()
	return &MongoDbAddon{masterNode: masterNode, communicator: communicator}
}

func init() {
	addAddon(NewMongoDbAddon)
}

//Name returns the addons name
func (addon MongoDbAddon) Name() string {
	return "mongodb"
}

//Requires returns a slice with the name of required addons
func (addon MongoDbAddon) Requires() []string {
	return []string{"openebs"}
}

//Description returns the addons description
func (addon MongoDbAddon) Description() string {
	return "MongoDB ReplicaSet (requires openebs addon. OpenEBS must be installed first)"
}

//URL returns the URL of the addons underlying project
func (addon MongoDbAddon) URL() string {
	return "https://www.mongodb.com/"
}

//Install performs all steps to install the addon
func (addon MongoDbAddon) Install(args ...string) {
	node := *addon.masterNode

	_, err := addon.communicator.RunCmd(node, "kubectl apply -f https://raw.githubusercontent.com/openebs/openebs/master/k8s/demo/mongodb/mongo-statefulset.yml")
	FatalOnError(err)

	improvedStorageClass := `---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: openebs-mongodb
provisioner: openebs.io/provisioner-iscsi
parameters:
  openebs.io/storage-pool: "default"
  openebs.io/jiva-replica-count: "3"
  openebs.io/volume-monitor: "true"
  openebs.io/capacity: 5G
  openebs.io/fstype: "xfs"
---`
	err = addon.communicator.WriteFile(node, "/root/openebs-mongodb-storageclass.yaml", improvedStorageClass, false)
	FatalOnError(err)

	_, err = addon.communicator.RunCmd(node, "kubectl apply -f openebs-mongodb-storageclass.yaml")
	FatalOnError(err)



	fmt.Println("MongoDB installed")
}

//Uninstall performs all steps to remove the addon
func (addon MongoDbAddon) Uninstall() {
	node := *addon.masterNode

	_, err := addon.communicator.RunCmd(node, "kubectl delete -f https://raw.githubusercontent.com/openebs/openebs/master/k8s/demo/mongodb/mongo-statefulset.yml")
	FatalOnError(err)

	fmt.Println("MongoDB removed")
}
