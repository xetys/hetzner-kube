package clustermanager

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

const maxErrors = 3

// K8sVersion is the version that will be used to install kubernetes
var K8sVersion = flag.String("k8s-version", "1.9.6-00",
	"The version of the k8s debian packages that will be used during provisioning")

// NodeProvisioner provisions all basic packages to install docker, kubernetes and wireguard
type NodeProvisioner struct {
	node         Node
	communicator NodeCommunicator
	eventService EventService
}

// NewNodeProvisioner creates a NodeProvisioner instance
func NewNodeProvisioner(node Node, communicator NodeCommunicator, eventService EventService) *NodeProvisioner {
	return &NodeProvisioner{
		node:         node,
		communicator: communicator,
		eventService: eventService,
	}
}

// Provision performs all steps to provision a node
func (provisioner *NodeProvisioner) Provision(node Node, communicator NodeCommunicator, eventService EventService) error {
	var err error
	errorCount := 0

	for !provisioner.packagesAreInstalled(node, communicator) {

		for err := provisioner.prepareAndInstall(); err != nil; {
			errorCount++

			if errorCount > maxErrors {
				return err
			}
		}

	}

	if err != nil {
		return err
	}

	eventService.AddEvent(node.Name, "packages installed")
	return nil
}

func (provisioner *NodeProvisioner) packagesAreInstalled(node Node, communicator NodeCommunicator) bool {
	out, err := communicator.RunCmd(node, "type -p kubeadm > /dev/null &> /dev/null; echo $?")
	if err != nil {
		return false
	}

	if strings.TrimSpace(out) == "0" {
		return true
	}
	return false
}

func (provisioner *NodeProvisioner) prepareAndInstall() error {

	err := provisioner.installTransportTools()
	if err != nil {
		return err
	}
	err = provisioner.preparePackages()
	if err != nil {
		return err
	}
	err = provisioner.prepareNetwork()
	if err != nil {
		return err
	}
	err = provisioner.updateAndInstall()
	return err
}

func (provisioner *NodeProvisioner) installTransportTools() error {

	provisioner.eventService.AddEvent(provisioner.node.Name, "installing transport tools")
	var err error
	for i := 0; i < 10; i++ {
		time.Sleep(3 * time.Second)
		_, err = provisioner.communicator.RunCmd(provisioner.node, "apt-get update && apt-get install -y apt-transport-https ca-certificates curl software-properties-common")
	}
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) preparePackages() error {
	provisioner.eventService.AddEvent(provisioner.node.Name, "prepare packages")

	err := provisioner.prepareDocker()
	if err != nil {
		return err
	}

	err = provisioner.prepareKubernetes()
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) prepareKubernetes() error {
	// kubernetes
	_, err := provisioner.communicator.RunCmd(provisioner.node, "curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -")
	if err != nil {
		return err
	}

	err = provisioner.communicator.WriteFile(provisioner.node, "/etc/apt/sources.list.d/kubernetes.list", `deb http://apt.kubernetes.io/ kubernetes-xenial main`, false)
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) prepareDocker() error {
	// docker-ce
	aptPreferencesDocker := `
Package: docker-ce
Pin: version 17.03.*
Pin-Priority: 1000
	`
	err := provisioner.communicator.WriteFile(provisioner.node, "/etc/apt/preferences.d/docker-ce", aptPreferencesDocker, false)
	if err != nil {
		return err
	}

	_, err = provisioner.communicator.RunCmd(provisioner.node, "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -")
	if err != nil {
		return err
	}

	_, err = provisioner.communicator.RunCmd(provisioner.node, `add-apt-repository "deb https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") $(lsb_release -cs) stable"`)
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) prepareNetwork() error {

	provisioner.eventService.AddEvent(provisioner.node.Name, "prepare network")

	err := provisioner.prepareFlannel()
	if err != nil {
		return err
	}

	err = provisioner.prepareWireguard()
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) prepareFlannel() error {
	// udev action to run systemd service on each flannel interface add
	flannelUdevRules :=
		`SUBSYSTEM=="net", ACTION=="add", KERNEL=="flannel.*", TAG+="systemd", ENV{SYSTEMD_WANTS}="flannel-created@%k.service"
`
	// systemd oneshot unit to run ethtool on corresponding interface
	flannelSystemd :=
		`[Unit]
Description=Disable TX checksum offload on flannel interface
[Service]
Type=oneshot
ExecStart=/sbin/ethtool -K %I tx off
`
	err := provisioner.communicator.WriteFile(provisioner.node, "/etc/udev/rules.d/71-flannel.rules", flannelUdevRules, false)
	if err != nil {
		return err
	}

	err = provisioner.communicator.WriteFile(provisioner.node, "/etc/systemd/system/flannel-created@.service", flannelSystemd, false)
	if err != nil {
		return err
	}

	_, err = provisioner.communicator.RunCmd(provisioner.node, "systemctl daemon-reload; systemctl restart systemd-udevd.service")
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) prepareWireguard() error {

	_, err := provisioner.communicator.RunCmd(provisioner.node, "add-apt-repository ppa:wireguard/wireguard -y")
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) updateAndInstall() error {
	provisioner.eventService.AddEvent(provisioner.node.Name, "updating packages")
	_, err := provisioner.communicator.RunCmd(provisioner.node, "apt-get update")
	if err != nil {
		return err
	}

	provisioner.eventService.AddEvent(provisioner.node.Name, "installing packages")
	command := fmt.Sprintf("apt-get install -y docker-ce kubelet=%s kubeadm=%s kubectl=%s wireguard ethtool linux-headers-$(uname -r) linux-headers-virtual",
		*K8sVersion, *K8sVersion, *K8sVersion)
	_, err = provisioner.communicator.RunCmd(provisioner.node, command)
	if err != nil {
		return err
	}

	return nil
}
