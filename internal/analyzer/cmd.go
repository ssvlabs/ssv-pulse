package analyzer

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabsinfra/ssv-pulse/configs"
	"github.com/ssvlabsinfra/ssv-pulse/internal/platform/cmd"
)

const filePathFlag = "log-file-path"

func init() {
	cmd.AddPersistentStringFlag(CMD, "logFilePath", "", "Path to ssv node log file to analyze", true)
	cmd.AddPersistentStringSliceFlag(CMD, "operators", []string{}, "Operators to analyze", false)
	cmd.AddPersistentBoolFlag(CMD, "cluster", false, "Are operators forming the cluster?", false)
}

var CMD = &cobra.Command{
	Use:   "log-analyzer",
	Short: "Read and analyze ssv node logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		isValid, err := configs.Values.Analyzer.Validate()
		if !isValid {
			panic(err.Error())
		}
		logFilePath := viper.GetString("logFilePath")
		if strings.Contains(logFilePath, "../") {
			return fmt.Errorf("❕ flag should not contain traversal")
		}
		if logFilePath == "" {
			return fmt.Errorf("❕ logFilePath flag can not be empty")
		}
		if err := viper.BindPFlag("cluster", cmd.PersistentFlags().Lookup("cluster")); err != nil {
			return err
		}
		cluster := viper.GetBool("cluster")
		if err := viper.BindPFlag("operators", cmd.PersistentFlags().Lookup("operators")); err != nil {
			return err
		}
		operators := viper.GetStringSlice("operators")
		analyzer, err := New(logFilePath, operators, cluster)
		if err != nil {
			return err
		}
		_, err = analyzer.AnalyzeConsensus()
		if err != nil {
			return err
		}
		return nil
	},
}
