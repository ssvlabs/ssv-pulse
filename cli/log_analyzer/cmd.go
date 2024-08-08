package log_analyzer_cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabsinfra/ssv-benchmark/cli/flags"
	"github.com/ssvlabsinfra/ssv-benchmark/src/log_analyzer"
)

func init() {
	flags.AddPersistentStringFlag(Run, "logFilePath", "", "Path to ssv node log file to analyze", true)
}

var Run = &cobra.Command{
	Use:   "log-analyzer",
	Short: "Read and analyze ssv node logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("logFilePath", cmd.PersistentFlags().Lookup("logFilePath")); err != nil {
			return err
		}
		logFilePath := viper.GetString("logFilePath")
		analyzer, err := log_analyzer.New(logFilePath)
		if err != nil {
			return nil
		}
		if err = analyzer.AnalyzeConsensus(); err != nil {
			return err
		}
		return nil
	},
}
