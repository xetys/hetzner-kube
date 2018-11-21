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
}
