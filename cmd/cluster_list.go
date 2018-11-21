package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// clusterListCmd represents the clusterList command
var clusterListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "lists all created clusters",
	Run: func(cmd *cobra.Command, args []string) {
		tw := new(tabwriter.Writer)
		tw.Init(os.Stdout, 0, 8, 2, '\t', 0)
		fmt.Fprintln(tw, "NAME\tNODES\tMASTER IP")

		for _, cluster := range AppConf.Config.Clusters {
			nodes := len(cluster.Nodes)
			var masterIP string
			for _, node := range cluster.Nodes {
				if node.IsMaster {
					masterIP = node.IPAddress
					break
				}
			}
			fmt.Fprintf(tw, "%s\t%d\t%s", cluster.Name, nodes, masterIP)
			fmt.Fprintln(tw)
		}

		tw.Flush()
	},
}

func init() {
	clusterCmd.AddCommand(clusterListCmd)
}
