package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version The current version of hetzner-kube.
var version = "DEVELOP"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "prints the current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
