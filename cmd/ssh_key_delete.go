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

	"errors"
	"github.com/spf13/cobra"
	"log"
)

// sshKeyDeleteCmd represents the sshKeyDelete command
var sshKeyDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "removes a saved SSH key from local configuration and Hetzner Cloud account",
	PreRunE: validateSHHKeyDeleteFlags,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		sshKey, _, err := AppConf.Client.SSHKey.Get(AppConf.Context, name)

		if err != nil {
			log.Fatal(err)
		}

		if sshKey == nil {
			log.Printf("SSH key not found: %s", name)
		} else {
			_, err = AppConf.Client.SSHKey.Delete(AppConf.Context, sshKey)
			FatalOnError(err)
		}

		if err = AppConf.Config.DeleteSSHKey(name); err != nil {
			log.Fatal(err)
		}

		AppConf.Config.WriteCurrentConfig()

		fmt.Println("SSH key deleted!")
	},
}

func validateSHHKeyDeleteFlags(cmd *cobra.Command, args []string) error {
	if err := AppConf.assertActiveContext(); err != nil {
		return err
	}

	if name, _ := cmd.Flags().GetString("name"); name == "" {
		return errors.New("flag --name is required")
	}

	return nil
}
func init() {
	sshKeyCmd.AddCommand(sshKeyDeleteCmd)
	sshKeyDeleteCmd.Flags().StringP("name", "n", "", "Name of the ssh-key")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sshKeyDeleteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sshKeyDeleteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
