package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <NAME>",
	Short: "adds a new context",
	Long: `This command adds a new context for communication with the Hetzner Cloud API.

	Before the context is actually saved, hetzner-kube ensures it can access the API using the token.
	On success, the newly added context is automatically used. Use the "context use" command, to switch contexts.
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		token, err := cmd.Flags().GetString("token")
		FatalOnError(err)

		if token == "" {
			r := bufio.NewReader(os.Stdin)
			for {
				fmt.Printf("Token: ")
				t, err := r.ReadString('\n')
				FatalOnError(err)
				t = strings.TrimSpace(t)
				if t == "" {
					continue
				}

				token = t
				break
			}
		}
		// test connection
		opts := []hcloud.ClientOption{
			hcloud.WithToken(token),
		}

		AppConf.Client = hcloud.NewClient(opts...)
		_, err = AppConf.Client.Server.All(AppConf.Context)

		FatalOnError(err)

		context := &HetznerContext{Name: name, Token: token}
		AppConf.Config.AddContext(*context)
		AppConf.Config.ActiveContextName = name
		AppConf.Config.WriteCurrentConfig()
		AppConf.CurrentContext = context
		fmt.Printf("added context '%s'", name)
	},
}

func init() {
	contextCmd.AddCommand(addCmd)

	addCmd.Flags().StringP("token", "t", "", "token of the context")
}
