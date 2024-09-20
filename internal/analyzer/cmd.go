package analyzer

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabs/ssv-pulse/configs"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/report"
)

const (
	logFilePathFlag = "log-file-path"
	operatorsFlag   = "operators"
	clusterFlag     = "cluster"

	command = "analyzer"
)

func init() {
	addFlags(CMD)
	if err := bindFlags(CMD); err != nil {
		panic(err.Error())
	}
}

var CMD = &cobra.Command{
	Use:   command,
	Short: "Read and analyze ssv node logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		isValid, err := configs.Values.Analyzer.Validate()
		if !isValid {
			return err
		}

		analyzer, err := New(configs.Values.Analyzer.LogFilePath, configs.Values.Analyzer.Operators, configs.Values.Analyzer.Cluster)
		if err != nil {
			return err
		}
		reportService := report.New()

		result, err := analyzer.AnalyzeConsensus()
		if err != nil {
			return err
		}
		for _, r := range result {
			reportService.AddRecord(report.Record{
				OperatorID:            r.OperatorID,
				BeaconTimeAvg:         r.AttestationTimeAverage,
				BeaconTimeMoreThanSec: strconv.FormatUint(uint64(r.AttestationDelayCount), 10) + "/" + strconv.FormatUint(uint64(r.AttestationTimeCount), 10),
				Score:                 r.CommitSignerScore,
				CommitDelayTotal:      r.CommitTotalDelay,
				PrepareDelayAvg:       r.PrepareDelayAvg,
				PrepareHighestDelay:   r.PrepareHighestDelay,
				PrepareMoreThanSec:    strconv.FormatUint(uint64(r.PrepareDelayCount), 10) + "/" + strconv.FormatUint(uint64(r.PrepareCount), 10),
			})
		}

		reportService.Render()

		return nil
	},
}

func addFlags(cobraCMD *cobra.Command) {
	cobraCMD.Flags().String(logFilePathFlag, "", "Path to ssv node log file to analyze")
	cobraCMD.Flags().StringSlice(operatorsFlag, []string{}, "Operators to analyze")
	cobraCMD.Flags().Bool(clusterFlag, false, "Are operators forming the cluster?")
}

func bindFlags(cmd *cobra.Command) error {
	if err := viper.BindPFlag("analyzer.log-file-path", cmd.Flags().Lookup(logFilePathFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("analyzer.cluster", cmd.Flags().Lookup(clusterFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("analyzer.operators", cmd.Flags().Lookup(operatorsFlag)); err != nil {
		return err
	}

	return nil
}
