package clustermanager

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const maxErrors = 3

// NodeProvisioner provisions all basic packages to install docker, kubernetes and wireguard
type NodeProvisioner struct {
	clusterName string
	node        Node
	manager     *Manager
}

// NewNodeProvisioner creates a NodeProvisioner instance
func NewNodeProvisioner(node Node, manager *Manager) *NodeProvisioner {
	return &NodeProvisioner{
		clusterName: manager.clusterName,
		node:        node,
		//communicator: manager.nodeCommunicator,
		//eventService: manager.eventService,
		manager: manager,
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

	return provisioner.disableSwap()
}

func (provisioner *NodeProvisioner) packagesAreInstalled(node Node, communicator NodeCommunicator) bool {
	file := "/etc/systemd/system/multi-user.target.wants/rke2-server.service"
	if !provisioner.node.IsMaster {
		file = "/etc/systemd/system/multi-user.target.wants/rke2-agent.service"
	}
	out, err := communicator.RunCmd(node, fmt.Sprintf("test -f %s; echo $?", file))
	if err != nil {
		return false
	}

	if strings.TrimSpace(out) == "0" {
		return true
	}
	return false
}

func (provisioner *NodeProvisioner) prepareAndInstall() error {

	err := provisioner.waitForCloudInitCompletion()
	if err != nil {
		return err
	}
	err = provisioner.installPackages()
	if err != nil {
		return err
	}
	//err = provisioner.preparePackages()
	//if err != nil {
	//	return err
	//}
	err = provisioner.installRke()
	if err != nil {
		return err
	}
	err = provisioner.setSystemWideEnvironment()
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) disableSwap() error {
	provisioner.manager.eventService.AddEvent(provisioner.node.Name, "disabling swap")

	_, err := provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, "swapoff -a")
	if err != nil {
		return err
	}

	_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, "sed -i '/ swap / s/^/#/' /etc/fstab")
	return err
}

func (provisioner *NodeProvisioner) waitForCloudInitCompletion() error {

	provisioner.manager.eventService.AddEvent(provisioner.node.Name, "waiting for cloud-init completion")
	var err error

	// define smal bash script to check if /var/lib/cloud/instance/boot-finished exist
	// this file created only when cloud-init finished its tasks
	cloudInitScript := `
#!/bin/bash

# timout is 10 min, return true immediately if ok, otherwise wait timout
# if cloud-init not very complex usually takes 2-3 min to completion
for i in {1..200}
do
  if [ -f /var/lib/cloud/instance/boot-finished ]; then
    exit 0
  fi
  sleep 3
done
exit 127
    `

	err = provisioner.manager.nodeCommunicator.WriteFile(provisioner.node, "/root/cloud-init-status-check.sh", cloudInitScript, AllExecute)
	if err != nil {
		return err
	}

	for i := 0; i < 10; i++ {
		time.Sleep(3 * time.Second)
		_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, "/root/cloud-init-status-check.sh")
	}
	if err != nil {
		return err
	}

	// remove script when done
	_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, "rm -f /root/cloud-init-status-check.sh")
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) installPackages() error {
	packageList := "apt-transport-https ca-certificates software-properties-common"

	if provisioner.manager.WireguardEnabled {
		packageList += " wireguard-tools"
	}

	if provisioner.manager.haEnabled {
		packageList += " haproxy"
	}

	provisioner.manager.eventService.AddEvent(provisioner.node.Name, "installing common packages")
	var err error
	for i := 0; i < 10; i++ {
		time.Sleep(3 * time.Second)
		_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, fmt.Sprintf(
			"apt-get update && apt-get install -y %s",
			packageList,
		))
	}
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) preparePackages() error {
	provisioner.manager.eventService.AddEvent(provisioner.node.Name, "prepare packages")

	err := provisioner.prepareDocker()
	if err != nil {
		return err
	}

	err = provisioner.prepareKubernetes()
	if err != nil {
		return err
	}

	// Wireguard (built into Ubuntu 20.04 kernel already, tools are optional)
	_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, "apt install -y wireguard-tools")
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) prepareKubernetes() error {
	// kubernetes
	_, err := provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, "curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -")
	if err != nil {
		return err
	}

	// Repository doesn't have Ubuntu 20.04 (focal), but `kubernetes-xenial` works
	err = provisioner.manager.nodeCommunicator.WriteFile(provisioner.node, "/etc/apt/sources.list.d/kubernetes.list", `deb http://apt.kubernetes.io/ kubernetes-xenial main`, AllRead)
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) prepareDocker() error {
	// docker-ce
	aptPreferencesDocker := `
Package: docker-ce
Pin: version 19.03.13~3-0~ubuntu-focal
Pin-Priority: 1000
	`
	err := provisioner.manager.nodeCommunicator.WriteFile(provisioner.node, "/etc/apt/preferences.d/docker-ce", aptPreferencesDocker, AllRead)
	if err != nil {
		return err
	}

	_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, `curl -fsSL https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg | apt-key add -`)
	if err != nil {
		return err
	}

	_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, `add-apt-repository "deb https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") $(lsb_release -cs) stable"`)
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *NodeProvisioner) installRke() error {
	provisioner.manager.eventService.AddEvent(provisioner.node.Name, "updating packages")
	_, err := provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, "apt-get update")
	if err != nil {
		return err
	}

	rke2Type := "server"
	if !provisioner.node.IsMaster {
		rke2Type = "agent"
	}
	provisioner.manager.eventService.AddEvent(provisioner.node.Name, fmt.Sprintf("installing RKE2 (%s)", provisioner.manager.kubernetesVersion))
	command := fmt.Sprintf("curl -sfL https://get.rke2.io | INSTALL_RKE2_VERSION=%s INSTALL_RKE2_TYPE=\"%s\" sh -",
		provisioner.manager.kubernetesVersion, rke2Type)
	_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, command)
	if err != nil {
		return err
	}

	if provisioner.node.IsMaster {
		_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, "systemctl enable rke2-server.service")
	} else {
		_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, "systemctl enable rke2-agent.service")
	}
	if err != nil {
		return err
	}

	return nil
}

// Last step because otherwise we need create script to check if variables already set and replaces them
// As soon as it is last step we are ok to set them in basic way
func (provisioner *NodeProvisioner) setSystemWideEnvironment() error {
	provisioner.manager.eventService.AddEvent(provisioner.node.Name, "set environment variables")
	var err error

	// set HETZNER_KUBE_MASTER
	_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, fmt.Sprintf("echo \"HETZNER_KUBE_MASTER=%s\" >> /etc/environment", strconv.FormatBool(provisioner.node.IsMaster)))
	if err != nil {
		return err
	}

	// set HETZNER_KUBE_CLUSTER
	_, err = provisioner.manager.nodeCommunicator.RunCmd(provisioner.node, fmt.Sprintf("echo \"HETZNER_KUBE_CLUSTER=%s\" >> /etc/environment", provisioner.clusterName))
	if err != nil {
		return err
	}

	return nil
}
