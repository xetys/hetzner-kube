package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// sshKeyListCmd represents the sshKeyList command
var sshKeyListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "lists all saved SSH keys",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		tw := new(tabwriter.Writer)
		tw.Init(os.Stdout, 0, 8, 0, '\t', 0)
		fmt.Fprintln(tw, "NAME\t")

		for _, key := range AppConf.Config.SSHKeys {
			fmt.Fprintf(tw, "%s", key.Name)
			fmt.Fprintln(tw)
		}

		tw.Flush()
	},
}

func init() {
	sshKeyCmd.AddCommand(sshKeyListCmd)
}
