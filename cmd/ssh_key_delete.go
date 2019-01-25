package cmd

import (
	"errors"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// sshKeyDeleteCmd represents the sshKeyDelete command
var sshKeyDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "removes a saved SSH key from local configuration and Hetzner Cloud account",
	PreRunE: validateSHHKeyDeleteFlags,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		sshKey, _, err := AppConf.Client.SSHKey.Get(AppConf.Context, name)

		if err != nil {
			log.Fatal(err)
		}

		if sshKey == nil {
			log.Printf("SSH key not found: %s", name)
		} else {
			_, err = AppConf.Client.SSHKey.Delete(AppConf.Context, sshKey)
			FatalOnError(err)
		}

		if err = AppConf.Config.DeleteSSHKey(name); err != nil {
			log.Fatal(err)
		}

		AppConf.Config.WriteCurrentConfig()

		fmt.Println("SSH key deleted!")
	},
}

func validateSHHKeyDeleteFlags(cmd *cobra.Command, args []string) error {
	if err := AppConf.assertActiveContext(); err != nil {
		return err
	}

	if name, _ := cmd.Flags().GetString("name"); name == "" {
		return errors.New("flag --name is required")
	}

	return nil
}
func init() {
	sshKeyCmd.AddCommand(sshKeyDeleteCmd)

	sshKeyDeleteCmd.Flags().StringP("name", "n", "", "Name of the ssh-key")
}
