package clustermanager

import (
	"bytes"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// SSHKey represents a keypair with the paths to the keys
type SSHKey struct {
	Name           string `json:"name"`
	PrivateKeyPath string `json:"private_key_path"`
	PublicKeyPath  string `json:"public_key_path"`
}

// SSHCommunicator implements NodeCommunicator as a SSH client
type SSHCommunicator struct {
	sshKeys     []SSHKey
	passPhrases map[string][]byte
}

var _ NodeCommunicator = &SSHCommunicator{}

// NewSSHCommunicator creates an instance of SSHCommunicator
func NewSSHCommunicator(sshKeys []SSHKey) NodeCommunicator {
	return &SSHCommunicator{
		sshKeys:     sshKeys,
		passPhrases: make(map[string][]byte),
	}
}

// RunCmd runs a bash command on the given node
func (sshComm *SSHCommunicator) RunCmd(node Node, command string) (output string, err error) {
	session, connection, err := sshComm.newSession(node)
	defer connection.Close()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	err = session.Run(command)

	if err != nil {
		return "", fmt.Errorf("run failed\ncommand:%s\nstdout:%s\nstderr:%v", command, stdoutBuf.String(), err)
	}
	session.Close()
	return stdoutBuf.String(), nil
}

func (sshComm *SSHCommunicator) newSession(node Node) (*ssh.Session, *ssh.Client, error) {
	signer, err := sshComm.getPrivateSSHKey(node.SSHKeyName)

	if err != nil {
		return nil, nil, err
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
			//log.Printf("dial failed:%v", err)
			if try > 10 {
				return nil, nil, err
			}
		} else {
			break
		}
		time.Sleep(1 * time.Second)
	}
	// log.Println("Connected succeeded!")
	session, err := connection.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("session failed:%v", err)
	}

	return session, connection, nil
}

// WriteFile places a file at a given part from string. Permissions are 0644, or 0755 if executable true
func (sshComm *SSHCommunicator) WriteFile(node Node, filePath string, content string, executable bool) error {
	signer, err := sshComm.getPrivateSSHKey(node.SSHKeyName)

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
	if err != nil {
		log.Fatalf("session failed:%v", err)
	}
	defer session.Close()

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

// CopyFileOverNode copies a file from a node to another. Does not work with directories.
func (sshComm *SSHCommunicator) CopyFileOverNode(sourceNode Node, targetNode Node, filePath string) error {
	return sshComm.TransformFileOverNode(sourceNode, targetNode, filePath, nil)
}

// TransformFileOverNode works like CopyFileOverNode, with the addition of changing the file contents using a func(string) string function
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

// findPrivateKeyByName returns a SSH key from its store
func (sshComm *SSHCommunicator) findPrivateKeyByName(name string) (int, *SSHKey) {
	index := -1
	for i, v := range sshComm.sshKeys {
		if v.Name == name {
			index = i
			return index, &v
		}
	}
	return index, nil
}

// CapturePassphrase asks the user to enter a private keys passphrase
func (sshComm *SSHCommunicator) CapturePassphrase(sshKeyName string) error {
	index, privateKey := sshComm.findPrivateKeyByName(sshKeyName)
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

	fmt.Print("Enter passphrase for SSH key " + privateKey.PrivateKeyPath + ": ")
	text, err := terminal.ReadPassword(int(syscall.Stdin))

	if err != nil {
		return err
	}

	fmt.Print("\n")
	sshComm.passPhrases[privateKey.PrivateKeyPath] = text

	// check that the captured password is correct
	_, err = sshComm.getPrivateSSHKey(sshKeyName)
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
		return false, errors.New("SSH: no key found")

	}

	return strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED"), nil
}

func (sshComm *SSHCommunicator) getPrivateSSHKey(sshKeyName string) (ssh.Signer, error) {
	privateKey, isEncrypted, pemBytes, err := sshComm.getInfoFromPrivateSSHKey(sshKeyName)
	if err != nil {
		return nil, err
	}

	if isEncrypted {
		return sshComm.getSignerFromEncrypthedPrivateSSHKey(sshKeyName, privateKey, pemBytes)
	}

	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("parse key failed:%v", err)
	}

	return signer, err
}

func (sshComm *SSHCommunicator) getInfoFromPrivateSSHKey(sshKeyName string) (*SSHKey, bool, []byte, error) {
	index, privateKey := sshComm.findPrivateKeyByName(sshKeyName)
	if index < 0 {
		return nil, false, nil, fmt.Errorf("cound not find SSH key '%s'", sshKeyName)
	}

	isEncrypted, err := sshComm.isEncrypted(privateKey)

	if err != nil {
		return nil, false, nil, err
	}

	pemBytes, err := ioutil.ReadFile(privateKey.PrivateKeyPath)
	if err != nil {
		return nil, false, nil, err
	}

	return privateKey, isEncrypted, pemBytes, nil
}

func (sshComm *SSHCommunicator) getSignerFromEncrypthedPrivateSSHKey(sshKeyName string, privateKey *SSHKey, pemBytes []byte) (ssh.Signer, error) {
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
}
