package analyzer

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabsinfra/ssv-pulse/configs"
)

const filePathFlag = "log-file-path"

func init() {
	cmd.AddPersistentStringFlag(CMD, "logFilePath", "", "Path to ssv node log file to analyze", true)
	cmd.AddPersistentStringFlag(CMD, "cluster", "1,2,3,4", "Cluster to analyze", true)
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
		cluster := viper.GetStringSlice("cluster")
		if strings.Contains(logFilePath, "../") {
			return fmt.Errorf("❕ flag should not contain traversal")
		}
		if len(cluster) == 0 {
			return fmt.Errorf("❕ operator IDs at cluster flag can not be empty")
		}
		analyzer, err := New(logFilePath, cluster)
		if err != nil {
			return err
		}
		if err = analyzer.AnalyzeConsensus(); err != nil {
			return err
		}
		return nil
	},
}
