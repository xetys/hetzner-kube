package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
	"github.com/xetys/hetzner-kube/pkg/hetzner"
)

var backupCmd = &cobra.Command{
	Use:     "backup",
	Short:   "creates a backup of the etcd cluster. If no name is provided, a current datetime string is used",
	PreRunE: validateClusterInArgumentExists,
	Run: func(cmd *cobra.Command, args []string) {
		snapshotName, _ := cmd.Flags().GetString("snapshot-name")
		etcdManager := getEtcdManager(cmd, args)

		err := etcdManager.CreateSnapshot(snapshotName)

		if err != nil {
			fmt.Println(err)
		}
	},
}

// getEtcdManager returns an instance of a configured EtcdManager
func getEtcdManager(cmd *cobra.Command, args []string) *clustermanager.EtcdManager {
	name := args[0]
	_, cluster := AppConf.Config.FindClusterByName(name)
	provider := hetzner.NewHetznerProvider(AppConf.Context, AppConf.Client, *cluster, AppConf.CurrentContext.Token)
	return clustermanager.NewEtcdManager(provider, AppConf.SSHClient)
}

func init() {
	etcdCmd.AddCommand(backupCmd)
	backupCmd.Flags().StringP("snapshot-name", "s", "", "Name of the snapshot")
}
