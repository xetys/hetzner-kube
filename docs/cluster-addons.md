# Developing addons for hetzner-kube

As of 0.3 adding cluster addons is quite easy, and can happen without altering the core code.
You only need to implement the `ClusterAddon` interface and provide a `ClusterAddonInitializer` to
register your addon to hetzner-kube.


## Step 1: Implement the `ClusterAddon` interface

A cluster interface is described by the following interface:

```go
type ClusterAddon interface {
	Name() string
	Description() string
	URL() string
	Install(args ...string)
	Uninstall()
}
```

To implement an addon which installs a simple nginx deployment, the addon should be configured like this:


```go

type NginxAddon struct {
	masterNode   *clustermanager.Node
	communicator clustermanager.NodeCommunicator
}

func NewNginxAddon(provider clustermanager.ClusterProvider, communicator clustermanager.NodeCommunicator) ClusterAddon {
	masterNode, err := provider.GetMasterNode()
	FatalOnError(err)
	return &NginxAddon{masterNode: masterNode, communicator: communicator}
}

func (addon *NginxAddon) Name() string {
	return "nginx"
}

func (addon *NginxAddon) Requires() []string {
	return []string{}
}

func (addon *NginxAddon) Description() string {
	return "a simple nginx deployment"
}

func (addon *NginxAddon) URL() string {
	return ""
}

func (addon *NginxAddon) Install(args ...string) {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "kubectl run nginx --image nginx")
	FatalOnError(err)
	log.Println("nginx installed")
}

func (addon *NginxAddon) Uninstall() {
	node := *addon.masterNode
	_, err := addon.communicator.RunCmd(node, "kubectl delete deployment nginx")
	FatalOnError(err)
	log.Println("nginx uninstalled")
}

```

The type should have a field with the pointer to the master node (for running `kubectl` or `helm`) and a node communicator 
(most commonly a SSH client) to run commands. In `NewNginxAddon` an `ClusterProvider` and `NodeCommunicator` instances are injected
so you can use them for getting the master node. You can use different data given by provider and define a different
type for your addon with custom fields, as long the type still implements `ClusterAddon`.

## Step 2: Register your addon

In the same file you add these lines:

```go

func init() {
	addAddon(NewNginxAddon)
}

```

`addAddon` expects a `ClusterAddonInitializer`, which is a function with a provider and node communicator as parameter 
and returning a `ClusterAddon` instance, which is satisfied by `NewNginxAddon`.


## Test

If all done right, you should see now your addon by doing `./hetzner-kube cluster addon list`
