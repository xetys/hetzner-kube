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
	"text/tabwriter"
	"os"
	"fmt"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
	"github.com/xetys/hetzner-kube/pkg/addons"
)

// clusterAddonInstallCmd represents the clusterAddonInstall command
var clusterAddonListCmd = &cobra.Command{
	Use:   "list",
	Short: "list the currently available addons",
	Run: func(cmd *cobra.Command, args []string) {
		tw := new(tabwriter.Writer)
		tw.Init(os.Stdout, 0, 8, 0, '\t', 0)
		fmt.Fprintln(tw, "NAME\tREQUIRES\tDESCRIPTION\tURL")

		cluster := &clustermanager.Cluster{Nodes: []clustermanager.Node{clustermanager.Node{IsMaster: true}}}
		provider, _ := hetzner.ProviderAndManager(*cluster, AppConf.Client, AppConf.Context, AppConf.SSHClient, nil, AppConf.CurrentContext.Token)
		addonService := addons.NewClusterAddonService(provider, AppConf.SSHClient)
		for _, addon := range addonService.Addons() {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t", addon.Name(), "-", addon.Description(), addon.URL())
			fmt.Fprintln(tw)
		}

		tw.Flush()
	},
}

func init() {
	clusterAddonCmd.AddCommand(clusterAddonListCmd)
}
