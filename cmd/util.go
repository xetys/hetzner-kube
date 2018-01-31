package cmd

import (
	"context"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"time"
	"io/ioutil"
	"bytes"
	"log"
	"golang.org/x/crypto/ssh"
	"errors"
	"fmt"
)

func runCmd(node Node, command string) (output string, err error) {
	index, privateKey := AppConf.Config.FindSSHKeyByName(node.SSHKeyName)
	if index < 0 {
		return "", errors.New(fmt.Sprintf("cound not find SSH key '%s'", node.SSHKeyName))
	}

	pemBytes, err := ioutil.ReadFile(privateKey.PrivateKeyPath)
	if err != nil {
		return "", err
	}
	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return "", errors.New(fmt.Sprintf("parse key failed:%v",err))
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
		log.Printf(">%s", stdoutBuf.String())
		return "", errors.New(fmt.Sprintf("Run failed:%v",err))
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
