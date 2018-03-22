package clustermanager

import (
	"golang.org/x/crypto/ssh"
	"time"
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"syscall"
	"io/ioutil"
	"encoding/pem"
	"strings"
	"errors"
	"log"
	"path"
)

type SSHKey struct {
	Name           string `json:"name"`
	PrivateKeyPath string `json:"private_key_path"`
	PublicKeyPath  string `json:"public_key_path"`
}
type SSHCommunicator struct {
	sshKeys []SSHKey
	passPhrases map[string][]byte
}

var _ NodeCommunicator = &SSHCommunicator{}

func NewSSHCommunicator(sshKeys []SSHKey) NodeCommunicator {
	return &SSHCommunicator{
		sshKeys: sshKeys,
	}
}


func (sshComm *SSHCommunicator) RunCmd(node Node, command string) (output string, err error) {
	signer, err := sshComm.getPrivateSshKey(node.SSHKeyName)

	if err != nil {
		return "", err
	}

	config := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	var connection *ssh.Client
	for try := 0; ; try++ {
		connection, err = ssh.Dial("tcp", node.IPAddress+":22", config)
		if err != nil {
			log.Printf("dial failed:%v", err)
			if try > 10 {
				return "", err
			}
		} else {
			break
		}
		time.Sleep(1 * time.Second)
	}
	defer connection.Close()
	// log.Println("Connected succeeded!")
	session, err := connection.NewSession()
	if err != nil {
		log.Fatalf("session failed:%v", err)
	}
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	err = session.Run(command)

	if err != nil {
		log.Println(stderrBuf.String())
		log.Printf("> %s", command)
		log.Println()
		log.Printf("%s", stdoutBuf.String())
		return "", fmt.Errorf("run failed:%v", err)
	}
	// log.Println("Command execution succeeded!")
	session.Close()
	return stdoutBuf.String(), nil
}

func (sshComm *SSHCommunicator) WriteFile(node Node, filePath string, content string, executable bool) error {
	signer, err := sshComm.getPrivateSshKey(node.SSHKeyName)

	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	var connection *ssh.Client
	for try := 0; ; try++ {
		connection, err = ssh.Dial("tcp", node.IPAddress+":22", config)
		if err != nil {
			log.Printf("dial failed:%v", err)
			if try > 10 {
				return err
			}
		} else {
			break
		}
		time.Sleep(1 * time.Second)
	}
	defer connection.Close()
	// log.Println("Connected succeeded!")
	session, err := connection.NewSession()
	defer session.Close()
	if err != nil {
		log.Fatalf("session failed:%v", err)
	}
	permission := "C0644"
	if executable {
		permission = "C0755"
	}
	fileName := path.Base(filePath)
	dir := path.Dir(filePath)

	var stderrBuf bytes.Buffer
	session.Stderr = &stderrBuf

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		fmt.Fprintln(w, permission, len(content), fileName)
		fmt.Fprint(w, content)
		fmt.Fprint(w, "\x00")
	}()

	err = session.Run("/usr/bin/scp -t " + dir)
	if err != nil {
		fmt.Println(stderrBuf.String())
		log.Fatalf("write failed:%v", err.Error())
	}

	return nil
}

func (sshComm *SSHCommunicator) CopyFileOverNode(sourceNode Node, targetNode Node, filePath string) error {
	return sshComm.TransformFileOverNode(sourceNode, targetNode, filePath, nil)
}

func (sshComm *SSHCommunicator) TransformFileOverNode(sourceNode Node, targetNode Node, filePath string, manipulator func(string) string) error {
	// get the file
	fileContent, err := sshComm.RunCmd(sourceNode, "cat "+filePath)
	if err != nil {
		return err
	}

	if manipulator != nil {
		fileContent = manipulator(fileContent)
	}

	// write file
	err = sshComm.WriteFile(targetNode, filePath, fileContent, false)
	return err
}

func (sshComm *SSHCommunicator) FindPrivateKeyByName(name string) (int, *SSHKey) {
	index := -1
	for i, v := range sshComm.sshKeys {
		if v.Name == name {
			index = i
			return index, &v
		}
	}
	return index, nil
}


func (sshComm *SSHCommunicator) CapturePassphrase(sshKeyName string) error {
	index, privateKey := sshComm.FindPrivateKeyByName(sshKeyName)
	if index < 0 {
		return fmt.Errorf("could not find SSH key '%s'", sshKeyName)
	}

	encrypted, err := sshComm.isEncrypted(privateKey)

	if err != nil {
		return err
	}

	if !encrypted {
		return nil
	}

	fmt.Print("Enter passphrase for sshComm key " + privateKey.PrivateKeyPath + ": ")
	text, err := terminal.ReadPassword(int(syscall.Stdin))

	if err != nil {
		return err
	}

	fmt.Print("\n")
	sshComm.passPhrases[privateKey.PrivateKeyPath] = text

	// check that the captured password is correct
	_, err = sshComm.getPrivateSshKey(sshKeyName)
	if err != nil {
		delete(sshComm.passPhrases, privateKey.PrivateKeyPath)
	}

	return err
}

func (sshComm *SSHCommunicator) getPassphrase(privateKeyPath string) ([]byte, error) {
	if phrase, ok := sshComm.passPhrases[privateKeyPath]; ok {
		return phrase, nil
	}

	return nil, errors.New("passphrase not found")
}

func (sshComm *SSHCommunicator) isEncrypted(privateKey *SSHKey) (bool, error) {
	pemBytes, err := ioutil.ReadFile(privateKey.PrivateKeyPath)
	if err != nil {
		return false, err
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return false, errors.New("sshComm: no key found")

	}

	return strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED"), nil
}

func (sshComm *SSHCommunicator) getPrivateSshKey(sshKeyName string) (ssh.Signer, error) {
	index, privateKey := sshComm.FindPrivateKeyByName(sshKeyName)
	if index < 0 {
		return nil, fmt.Errorf("cound not find SSH key '%s'", sshKeyName)
	}

	encrypted, err := sshComm.isEncrypted(privateKey)

	if err != nil {
		return nil, err
	}

	pemBytes, err := ioutil.ReadFile(privateKey.PrivateKeyPath)
	if err != nil {
		return nil, err
	}

	if encrypted {
		passPhrase, err := sshComm.getPassphrase(privateKey.PrivateKeyPath)
		if err != nil {
			// Fallback as sometimes the cache with the passphrases is not set, i.e. on program start
			err = sshComm.CapturePassphrase(sshKeyName)
			if err != nil {
				return nil, fmt.Errorf("error capturing passphrase:%v", err)
			}
			passPhrase, err = sshComm.getPassphrase(privateKey.PrivateKeyPath)
		}
		if err != nil {
			return nil, fmt.Errorf("parse key failed:%v", err)
		}

		signer, err := ssh.ParsePrivateKeyWithPassphrase(pemBytes, passPhrase)
		if err != nil {
			return nil, fmt.Errorf("parse key failed:%v", err)
		}

		return signer, err
	} else {
		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("parse key failed:%v", err)
		}

		return signer, err
	}
}
