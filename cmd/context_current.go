package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// currentCmd represents the current command
var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "prints the current used context",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(AppConf.Config.ActiveContextName)
	},
}

func init() {
	contextCmd.AddCommand(currentCmd)
}
