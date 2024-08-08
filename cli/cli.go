package cli

import (
	"log"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	loganalyzer "github.com/ssvlabsinfra/ssv-benchmark/cli/log_analyzer"
)

func init() {
	RootCmd.AddCommand(loganalyzer.Run)
}

var RootCmd = &cobra.Command{
	Use:   "ssv-analyzer",
	Short: "CLI for analyzing and benchmarking ssv node",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
	},
}

// Execute executes the root command
func Execute(appName, version string) {
	RootCmd.Short = appName
	RootCmd.Version = version
	loganalyzer.Run.Version = version
	if err := RootCmd.Execute(); err != nil {
		log.Fatal("failed to execute root command", zap.Error(err))
	}
}
