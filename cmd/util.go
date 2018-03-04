package cmd

import (
	"bytes"
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/Pallinder/go-randomdata"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"syscall"
	"time"
)

var sshPassPhrases = make(map[string][]byte)

func capturePassphrase(sshKeyName string) error {
	index, privateKey := AppConf.Config.FindSSHKeyByName(sshKeyName)
	if index < 0 {
		return errors.New(fmt.Sprintf("could not find SSH key '%s'", sshKeyName))
	}

	encrypted, err := isEncrypted(privateKey)

	if err != nil {
		return err
	}

	if !encrypted {
		return nil
	}

	fmt.Print("Enter passphrase for ssh key " + privateKey.PrivateKeyPath + ": ")
	text, err := terminal.ReadPassword(int(syscall.Stdin))

	if err != nil {
		return err
	}

	fmt.Print("\n")
	sshPassPhrases[privateKey.PrivateKeyPath] = text

	return nil
}

func getPassphrase(privateKeyPath string) ([]byte, error) {
	if phrase, ok := sshPassPhrases[privateKeyPath]; ok {
		return phrase, nil
	}

	return nil, errors.New("passphrase not found")
}

func isEncrypted(privateKey *SSHKey) (bool, error) {
	pemBytes, err := ioutil.ReadFile(privateKey.PrivateKeyPath)
	if err != nil {
		return false, err
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return false, errors.New("ssh: no key found")

	}

	return strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED"), nil
}

func getPrivateSshKey(sshKeyName string) (ssh.Signer, error) {
	index, privateKey := AppConf.Config.FindSSHKeyByName(sshKeyName)
	if index < 0 {
		return nil, errors.New(fmt.Sprintf("cound not find SSH key '%s'", sshKeyName))
	}

	encrypted, err := isEncrypted(privateKey)

	if err != nil {
		return nil, err
	}

	pemBytes, err := ioutil.ReadFile(privateKey.PrivateKeyPath)
	if err != nil {
		return nil, err
	}

	if encrypted {
		passPhrase, err := getPassphrase(privateKey.PrivateKeyPath)
		if err != nil {
			// Fallback as sometimes the cache with the passphrases is not set, i.e. on program start
			err = capturePassphrase(sshKeyName)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("error capturing passphrase:%v", err))
			}
			passPhrase, err = getPassphrase(privateKey.PrivateKeyPath)
		}
		if err != nil {
			return nil, errors.New(fmt.Sprintf("parse key failed:%v", err))
		}

		signer, err := ssh.ParsePrivateKeyWithPassphrase(pemBytes, passPhrase)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("parse key failed:%v", err))
		}

		return signer, err
	} else {
		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("parse key failed:%v", err))
		}

		return signer, err
	}
}

func writeNodeFile(node Node, filePath string, content string, executable bool) error {
	signer, err := getPrivateSshKey(node.SSHKeyName)

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

func copyFileOverNode(sourceNode Node, targetNode Node, filePath string, manipulator func(string) string) error {
	// get the file
	fileContent, err := runCmd(sourceNode, "cat "+filePath)
	if err != nil {
		return err
	}

	if manipulator != nil {
		fileContent = manipulator(fileContent)
	}

	// write file
	err = writeNodeFile(targetNode, filePath, fileContent, false)
	return err
}

func runCmd(node Node, command string) (output string, err error) {
	signer, err := getPrivateSshKey(node.SSHKeyName)

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
		return "", errors.New(fmt.Sprintf("Run failed:%v", err))
	}
	// log.Println("Command execution succeeded!")
	session.Close()
	return stdoutBuf.String(), nil
}

func waitAction(ctx context.Context, client *hcloud.Client, action *hcloud.Action) (<-chan error, <-chan int) {
	errCh := make(chan error, 1)
	progressCh := make(chan int)

	go func() {
		defer close(errCh)
		defer close(progressCh)

		ticker := time.NewTicker(100 * time.Millisecond)

		sendProgress := func(p int) {
			select {
			case progressCh <- p:
				break
			default:
				break
			}
		}

		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case <-ticker.C:
				break
			}

			action, _, err := client.Action.GetByID(ctx, action.ID)
			if err != nil {
				errCh <- ctx.Err()
				return
			}

			switch action.Status {
			case hcloud.ActionStatusRunning:
				sendProgress(action.Progress)
				break
			case hcloud.ActionStatusSuccess:
				sendProgress(100)
				errCh <- nil
				return
			case hcloud.ActionStatusError:
				errCh <- action.Error()
				return
			}
		}
	}()

	return errCh, progressCh
}

func randomName() string {
	return fmt.Sprintf("%s-%s%s", randomdata.Adjective(), randomdata.Noun(), randomdata.Adjective())
}

func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

func Include(vs []string, t string) bool {
	return Index(vs, t) >= 0
}

func FatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func waitOrError(tc chan bool, ec chan error, numProcPtr *int) error {
	numProcs := *numProcPtr
	for numProcs > 0 {
		select {
		case err := <-ec:
			return err
		case <-tc:
			numProcs--
		}
	}

	return nil
}
