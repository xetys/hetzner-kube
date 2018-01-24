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
	"bufio"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <NAME>",
	Short: "adds a new context",
	Long: `This command adds a new context for communication with the Hetzner Cloud API.

	Before the context is actually saved, hetzner-kube ensures it can access the API using the token.
	On success, the newly added context is automatically used. Use the "context use" command, to switch contexts.
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name, token := args[0], ""
		r := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("Token: ")
			t, err := r.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}

			token = t
			break
		}

		// test connection
		opts := []hcloud.ClientOption{
			hcloud.WithToken(token),
		}

		AppConf.Client = hcloud.NewClient(opts...)
		_, err := AppConf.Client.Server.All(AppConf.Context)

		if err != nil {
			log.Fatal(err)
		}

		context := &HetznerContext{Name: name, Token: token}
		AppConf.Config.AddContext(*context)
		AppConf.Config.ActiveContextName = name
		AppConf.Config.WriteCurrentConfig()
		AppConf.CurrentContext = context
		fmt.Printf("added context '%s'", name)
	},
}

func init() {
	contextCmd.AddCommand(addCmd)

	addCmd.Flags().StringP("token", "t", "", "token of the context")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
