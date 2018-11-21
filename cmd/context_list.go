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
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "lists all contexts",
	Long:    `A context is defined by a name and an API token to access the public Hetzner Cloud API.`,
	Run: func(cmd *cobra.Command, args []string) {
		tw := new(tabwriter.Writer)
		tw.Init(os.Stdout, 0, 8, 0, '\t', 0)
		fmt.Fprintln(tw, "NAME\tTOKEN")

		for _, context := range AppConf.Config.Contexts {
			fmt.Fprintf(tw, "%s\t%s\t", context.Name, context.Token)
			fmt.Fprintln(tw)
		}

		tw.Flush()
	},
}

func init() {
	contextCmd.AddCommand(listCmd)
}
