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
	"github.com/xetys/hetzner-kube/pkg/hetzner"
	"github.com/xetys/hetzner-kube/pkg/addons"
)

// clusterAddonInstallCmd represents the clusterAddonInstall command
var clusterAddonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "removes an addon to a cluster",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return nil
		}

		if name == "" {
			return errors.New("flag --name is required")
		}

		idx, _ := AppConf.Config.FindClusterByName(name)

		if idx == -1 {
			return fmt.Errorf("cluster '%s' not found", name)
		}
		if len(args) != 1 {
			return errors.New("exactly one argument expected")
		}
		addonName := args[0]
		if !AddonExists(addonName) {
			return fmt.Errorf("addon %s not found", addonName)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		addonName := args[0]

		_, cluster := AppConf.Config.FindClusterByName(name)

		log.Printf("removing addon %s", addonName)
		provider, _ := hetzner.ProviderAndManager(*cluster, AppConf.Client, AppConf.Context, AppConf.SSHClient, nil, AppConf.CurrentContext.Token)
		addonService := addons.NewClusterAddonService(provider, AppConf.SSHClient)
		addon := addonService.GetAddon(addonName)
		addon.Uninstall()

		log.Printf("addon %s successfully removed", addonName)
	},
}

func init() {
	clusterAddonCmd.AddCommand(clusterAddonUninstallCmd)
	clusterAddonUninstallCmd.Flags().StringP("name", "n", "", "Name of the cluster")
}
