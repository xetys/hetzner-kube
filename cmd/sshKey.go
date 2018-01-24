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
	"github.com/spf13/cobra"
)

// sshKeyCmd represents the sshKey command
var sshKeyCmd = &cobra.Command{
	Use:   "ssh-key",
	Short: "view and manage SSH keys",
	Long: `This sub-command handles both, the public key entry in Hetzner Cloud and private key location of your machine.

Note, that the private key never gets uploaded anywhere. The path is used to connect to the servers`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	rootCmd.AddCommand(sshKeyCmd)
	sshKeyCmd.Flags().StringP("name", "n", "", "")


	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sshKeyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sshKeyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
