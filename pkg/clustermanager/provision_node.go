package clustermanager

import "strings"

const MAX_ERRORS = 3

func Provision(node Node, communicator NodeCommunicator, eventService EventService) error {
	var err error = nil
	errorCount := 0

	for !PackagesAreInstalled(node, communicator) {
		aptPreferencesDocker := `
Package: docker-ce
Pin: version 17.03.*
Pin-Priority: 1000
	`
		err = communicator.WriteFile(node, "/etc/apt/preferences.d/docker-ce", aptPreferencesDocker, false)
		if err != nil {
			errorCount++
			if errorCount < MAX_ERRORS {
				continue
			}
			break
		}

		eventService.AddEvent(node.Name, "installing transport tools")
		_, err = communicator.RunCmd(node, "apt-get update && apt-get install -y apt-transport-https ca-certificates curl software-properties-common")
		if err != nil {
			errorCount++
			if errorCount < MAX_ERRORS {
				continue
			}
			break
		}


		eventService.AddEvent(node.Name, "prepare packages")

		// docker-ce
		_, err = communicator.RunCmd(node, "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -")
		if err != nil {
			errorCount++
			if errorCount < MAX_ERRORS {
				continue
			}
			break
		}

		// kubernetes
		_, err = communicator.RunCmd(node, "curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -")
		if err != nil {
			errorCount++
			if errorCount < MAX_ERRORS {
				continue
			}
			break
		}
		_, err = communicator.RunCmd(node, `add-apt-repository "deb https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") $(lsb_release -cs) stable"`)
		if err != nil {
			errorCount++
			if errorCount < MAX_ERRORS {
				continue
			}
			break
		}

		err = communicator.WriteFile(node, "/etc/apt/sources.list.d/kubernetes.list", `deb http://apt.kubernetes.io/ kubernetes-xenial main`, false)
		if err != nil {
			break
		}

		// wireguard
		_, err = communicator.RunCmd(node, "add-apt-repository ppa:wireguard/wireguard -y")
		if err != nil {
			errorCount++
			if errorCount < MAX_ERRORS {
				continue
			}
			return err
		}

		eventService.AddEvent(node.Name, "updating packages")
		_, err = communicator.RunCmd(node, "apt-get update")
		if err != nil {
			errorCount++
			if errorCount < MAX_ERRORS {
				continue
			}
			break
		}

		eventService.AddEvent(node.Name, "installing packages")
		_, err = communicator.RunCmd(node, "apt-get install -y docker-ce kubelet kubeadm kubectl wireguard linux-headers-$(uname -r) linux-headers-virtual")
		if err != nil {
			errorCount++
			if errorCount < MAX_ERRORS {
				continue
			}
			break
		}

		_, err = communicator.RunCmd(node, "systemctl daemon-reload")
		if err != nil {
			errorCount++
			if errorCount < MAX_ERRORS {
				continue
			}
			break
		}
	}

	if err != nil {
		return err
	}

	eventService.AddEvent(node.Name, "packages installed")
	return nil
}

func PackagesAreInstalled(node Node, communicator NodeCommunicator) bool {
	out, err := communicator.RunCmd(node, "type -p kubeadm > /dev/null &> /dev/null; echo $?")
	if err != nil {
		return false
	}

	if strings.TrimSpace(out) == "0" {
		return true
	} else {
		return false
	}
}