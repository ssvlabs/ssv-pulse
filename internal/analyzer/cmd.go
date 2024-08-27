package analyzer

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabsinfra/ssv-pulse/configs"
)

const filePathFlag = "log-file-path"

func init() {
	CMD.Flags().String(filePathFlag, "", "Path to ssv node log file to analyze")
	if err := viper.BindPFlag("analyzer.log-file-path", CMD.Flags().Lookup(filePathFlag)); err != nil {
		panic(err.Error())
	}
}

var CMD = &cobra.Command{
	Use:   "log-analyzer",
	Short: "Read and analyze ssv node logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		isValid, err := configs.Values.Analyzer.Validate()
		if !isValid {
			panic(err.Error())
		}

		analyzer, err := New(configs.Values.Analyzer.LogFilePath)
		if err != nil {
			return nil
		}
		if err = analyzer.AnalyzeConsensus(); err != nil {
			return err
		}
		return nil
	},
}
