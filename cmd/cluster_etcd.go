package cmd

import "github.com/spf13/cobra"

var etcdCmd = &cobra.Command{
	Use:   "etcd",
	Short: "a subcommand for performing etcd tasks",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	clusterCmd.AddCommand(etcdCmd)
}
