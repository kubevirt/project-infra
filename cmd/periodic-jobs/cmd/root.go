package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "periodic-jobs",
		Short: "periodic-jobs provides tools for managing Prow periodic job schedules",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		},
	}
	inputFile string
	pattern   string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&inputFile, "input",
		"github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml",
		"Input YAML file path")
	rootCmd.PersistentFlags().StringVar(&pattern, "pattern",
		"periodic-kubevirt-e2e-k8s-",
		"Job name prefix to match")

	rootCmd.AddCommand(GanttCommand())
	rootCmd.AddCommand(SpreadCommand())
}

func Execute() error {
	return rootCmd.Execute()
}
