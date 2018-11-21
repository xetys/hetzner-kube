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
