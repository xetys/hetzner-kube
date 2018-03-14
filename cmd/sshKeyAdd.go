// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"bytes"
	"errors"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

// sshKeyAddCmd represents the sshKeyAdd command
var sshKeyAddCmd = &cobra.Command{
	Use:   "add",
	Short: "adds a new SSH key to the Hetzner Cloud project and local configuration",
	Long: `This sub-command saves the path of the provided SSH private key in a configuration file on your local machine.
Then it uploads it corresponding public key with the provided name to the Hetzner Cloud project, associated by the current context.

Note: the private key is never uploaded to any server at any time.`,
	PreRunE: validateFlags,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("sshKeyAdd called")
		name, _ := cmd.Flags().GetString("name")
		publicKeyPath, _ := cmd.Flags().GetString("public-key-path")
		privateKeyPath, _ := cmd.Flags().GetString("private-key-path")

		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		privateKeyPath = strings.Replace(privateKeyPath, "~", home, 1)
		publicKeyPath = strings.Replace(publicKeyPath, "~", home, 1)

		var (
			data []byte
		)
		if publicKeyPath == "-" {
			data, err = ioutil.ReadAll(os.Stdin)
		} else {
			data, err = ioutil.ReadFile(publicKeyPath)
		}
		if err != nil {
			log.Fatalln(err)
		}
		publicKey := string(data)

		opts := hcloud.SSHKeyCreateOpts{
			Name:      name,
			PublicKey: publicKey,
		}

		context := AppConf.Context
		client := AppConf.Client
		sshKey, res, err := client.SSHKey.Create(context, opts)

		if res.StatusCode == http.StatusConflict {
			pkey, _, _, _, err := ssh.ParseAuthorizedKey(data)
			if err != nil {
				log.Fatalln(err)
			}
			// check if the key is already in to local app config
			for _, sshKey := range AppConf.Config.SSHKeys {
				localData, err := ioutil.ReadFile(sshKey.PublicKeyPath)
				if err != nil {
					log.Fatalln(err)
				}
				localPkey, _, _, _, err := ssh.ParseAuthorizedKey(localData)
				if err != nil {
					log.Fatalln(err)
				}
				// if the key is in the local app config print a message and return
				if bytes.Equal(pkey.Marshal(), localPkey.Marshal()) {
					fmt.Printf("SSH key does already exists in your config as %s\n", sshKey.Name)
					return
				}
			}
			// if the key is not in the local app config, fetch it from hetzner
			sshKeys, err := client.SSHKey.All(context)
			if err != nil {
				log.Fatalln(err)
			}
			for _, sshKeyHetzner := range sshKeys {
				hetznerPkey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(sshKeyHetzner.PublicKey))
				if err != nil {
					log.Fatalln(err)
				}
				if bytes.Equal(pkey.Marshal(), hetznerPkey.Marshal()) {
					fmt.Printf("SSH key does already on hetzner as '%s'\n", sshKeyHetzner.Name)
					fmt.Printf("SSH key will be added to your config as '%s'\n", sshKeyHetzner.Name)
					// We replace the failed request response with the fetched sshkey that has the same public key
					sshKey = sshKeyHetzner
					break
				}
				if sshKeyHetzner.Name == name {
					log.Fatalf("Name '%s' is already taken!", name)
				}
			}
		} else if err != nil {
			log.Fatalln(err)
		}

		AppConf.Config.AddSSHKey(clustermanager.SSHKey{
			Name:           sshKey.Name,
			PrivateKeyPath: privateKeyPath,
			PublicKeyPath:  publicKeyPath,
		})

		AppConf.Config.WriteCurrentConfig()

		fmt.Printf("SSH key %s(%d) created\n", sshKey.Name, sshKey.ID)
	},
}

func validateFlags(cmd *cobra.Command, args []string) error {
	if err := AppConf.assertActiveContext(); err != nil {
		return err
	}

	if name, _ := cmd.Flags().GetString("name"); name == "" {
		return errors.New("flag --name is required")
	}

	privateKeyPath, _ := cmd.Flags().GetString("private-key-path")
	if privateKeyPath == "" {
		return errors.New("flag --private-key-path cannot be empty")
	}

	publicKeyPath, _ := cmd.Flags().GetString("public-key-path")
	if publicKeyPath == "" {
		return errors.New("flag --public-key-path cannot be empty")
	}

	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	privateKeyPath = strings.Replace(privateKeyPath, "~", home, 1)
	publicKeyPath = strings.Replace(publicKeyPath, "~", home, 1)
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("could not find private key '%s'", privateKeyPath)

	}

	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("could not find public key '%s'", publicKeyPath)
	}

	return nil
}

func init() {
	sshKeyCmd.AddCommand(sshKeyAddCmd)
	sshKeyAddCmd.Flags().StringP("name", "n", "", "the name of the key")
	sshKeyAddCmd.Flags().String("private-key-path", "~/.ssh/id_rsa", "the path to the private key")
	sshKeyAddCmd.Flags().String("public-key-path", "~/.ssh/id_rsa.pub", "the path to the public key")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sshKeyAddCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sshKeyAddCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
