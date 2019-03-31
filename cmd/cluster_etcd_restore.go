package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:     "restore",
	Short:   "restores an etcd snapshot",
	PreRunE: validateEtcdRestoreCmd,
	Run: func(cmd *cobra.Command, args []string) {
		snapshotName, _ := cmd.Flags().GetString("snapshot-name")
		skipDistribution, _ := cmd.Flags().GetBool("skip-distribution")

		etcdManager := getEtcdManager(cmd, args)
		succeeded, err := etcdManager.RestoreSnapshot(snapshotName, skipDistribution)

		if err != nil {
			fmt.Println(err)
		}

		if succeeded {
			fmt.Println("restore succeeded")
		}
	},
}

// validateEtcdRestoreCmd checks the required conditions to restore a snapshot
func validateEtcdRestoreCmd(cmd *cobra.Command, args []string) error {
	err := validateClusterInArgumentExists(cmd, args)

	if err != nil {
		return err
	}

	snapshotName, err := cmd.Flags().GetString("snapshot-name")

	if err != nil {
		return err
	}

	if snapshotName == "" {
		return fmt.Errorf("--snapshot-name or -s should not be empty")
	}

	return nil
}

func init() {
	etcdCmd.AddCommand(restoreCmd)
	restoreCmd.Flags().StringP("snapshot-name", "n", "", "Name of the snapshot")
	restoreCmd.Flags().BoolP("skip-distribution", "s", false, "skips distribution of snapshots. Useful, if performing restore on a previously restored snapshot")
}
