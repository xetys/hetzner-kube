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
	"errors"
)

// clusterAddWorkerCmd represents the clusterAddWorker command
var cloudInitSetCmd = &cobra.Command{
	Use:   "cloud-init",
	Short: "set cloud-init file",
	Long: `Sets the file used to preconfigure the newly created server.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		cifile, err := cmd.Flags().GetString("file")
		if err != nil {
			return nil
		}

		if len(cifile) == 0 {
			return errors.New("flag --file is required")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		cloudInitFile, _ := cmd.Flags().GetString("file")
		AppConf.Config.CloudInitFile = cloudInitFile
		AppConf.Config.WriteCurrentConfig()
	},
}

func init() {
	rootCmd.AddCommand(cloudInitSetCmd)
	cloudInitSetCmd.Flags().StringP("file", "f", "", "Cloud-init file for server preconfiguration")
}
