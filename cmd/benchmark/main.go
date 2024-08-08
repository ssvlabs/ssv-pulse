package main

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/ssvlabsinfra/ssv-benchmark/internal/analyzer"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/cmd"
	_ "github.com/ssvlabsinfra/ssv-benchmark/internal/platform/logger"
)

var (
	appName = "ssv-benchmark"
	version = "1.0"
)

func init() {
	rootCmd.AddCommand(analyzer.CMD)
	rootCmd.AddCommand(benchmark.CMD)
	rootCmd.AddCommand(cmd.Version)
}

var rootCmd = &cobra.Command{
	Use:   "ssv-benchmark",
	Short: "CLI for analyzing and benchmarking ssv node",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
	},
}

func main() {
	rootCmd.Short = appName
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		slog.Error("failed to execute root command")
		panic(err.Error())
	}
}
