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
	"log"

	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg/addons"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

// clusterAddonInstallCmd represents the clusterAddonInstall command
var clusterAddonInstallCmd = &cobra.Command{
	Use:     "install",
	Short:   "installs an addon to a cluster",
	PreRunE: validateAddonSubCommand,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		addonName := args[0]

		_, cluster := AppConf.Config.FindClusterByName(name)

		log.Printf("installing addon %s", addonName)
		provider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
		addonService := addons.NewClusterAddonService(provider, AppConf.SSHClient)
		masterNode, err := provider.GetMasterNode()
		FatalOnError(err)

		err = AppConf.SSHClient.(*clustermanager.SSHCommunicator).CapturePassphrase(masterNode.SSHKeyName)
		FatalOnError(err)

		addon := addonService.GetAddon(addonName)
		addon.Install(args...)

		log.Printf("addon %s successfully installed", addonName)
	},
}

func init() {
	clusterAddonCmd.AddCommand(clusterAddonInstallCmd)
	clusterAddonInstallCmd.Flags().StringP("name", "n", "", "Name of the cluster")
}
