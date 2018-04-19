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
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var contextDeleteCmd = &cobra.Command{
	Use:   "delete <NAME>",
	Short: "deletes a new context",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		err := AppConf.DeleteContextByName(name)
		FatalOnError(err)

		if name == AppConf.Config.ActiveContextName {
			for _, context := range AppConf.Config.Contexts {
				if context.Name != name {
					AppConf.Config.ActiveContextName = context.Name
					AppConf.CurrentContext = &context
					fmt.Printf("switched to context '%s'\n", context.Name)
					break
				}
			}
		}
		AppConf.Config.WriteCurrentConfig()
		fmt.Printf("deleted context '%s'\n", name)
	},
}

func init() {
	contextCmd.AddCommand(contextDeleteCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
