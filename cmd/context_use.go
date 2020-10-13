package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// useCmd represents the use command
var useCmd = &cobra.Command{
	Use:   "use <NAME>",
	Short: "switches to a saved context given by NAME",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextName := args[0]

		err := AppConf.SwitchContextByName(contextName)

		FatalOnError(err)

		AppConf.Config.WriteCurrentConfig()
		fmt.Printf("switched to context '%s'\n", contextName)
	},
}

func init() {
	contextCmd.AddCommand(useCmd)
}
