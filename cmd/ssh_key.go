package cmd

import (
	"github.com/spf13/cobra"
)

// sshKeyCmd represents the sshKey command
var sshKeyCmd = &cobra.Command{
	Use:   "ssh-key",
	Short: "view and manage SSH keys",
	Long: `This sub-command handles both, the public key entry in Hetzner Cloud and private key location of your machine.

Note, that the private key never gets uploaded anywhere. The path is used to connect to the servers`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	rootCmd.AddCommand(sshKeyCmd)

	sshKeyCmd.Flags().StringP("name", "n", "", "")
}
