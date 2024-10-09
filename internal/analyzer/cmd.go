package analyzer

import (
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabs/ssv-pulse/configs"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/attestation"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/commit"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/operator"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/prepare"
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

		attestationAnalyzer, err := attestation.New(
			configs.Values.Analyzer.LogFilePath,
			configs.Values.Analyzer.Operators,
			configs.Values.Analyzer.Cluster)
		if err != nil {
			return err
		}

		commitAnalyzer, err := commit.New(
			configs.Values.Analyzer.LogFilePath,
			configs.Values.Analyzer.Operators,
			configs.Values.Analyzer.Cluster)
		if err != nil {
			return err
		}

		prepareAnalyzer, err := prepare.New(
			configs.Values.Analyzer.LogFilePath,
			configs.Values.Analyzer.Operators,
			configs.Values.Analyzer.Cluster)
		if err != nil {
			return err
		}

		operatorAnalyzer, err := operator.New(
			configs.Values.Analyzer.LogFilePath)
		if err != nil {
			return err
		}

		analyzerSvc, err := New(
			operatorAnalyzer,
			attestationAnalyzer,
			commitAnalyzer,
			prepareAnalyzer,
			configs.Values.Analyzer.Operators,
			configs.Values.Analyzer.Cluster)

		if err != nil {
			return err
		}

		operatorReport := report.NewOperator(configs.Values.Analyzer.WithScores())
		consensusReport := report.NewConsensus()

		result, err := analyzerSvc.Start()
		if err != nil {
			return err
		}

		var (
			isSet                              bool
			consensusResponseTimeAvg           time.Duration
			consensusClientResponseTimeDelayed string
		)

		for _, r := range result.OperatorStats {
			operatorReport.AddRecord(report.OperatorRecord{
				OperatorID:     r.OperatorID,
				IsLogFileOwner: r.IsLogFileOwner,

				Score:               r.CommitSignerScore,
				CommitDelayTotal:    r.CommitTotalDelay,
				PrepareDelayAvg:     r.PrepareDelayAvg,
				PrepareHighestDelay: r.PrepareHighestDelay,
				PrepareMoreThanSec:  strconv.FormatUint(uint64(r.PrepareDelayCount), 10) + "/" + strconv.FormatUint(uint64(r.PrepareCount), 10),
			})

			if !isSet {
				consensusResponseTimeAvg = r.AttestationTimeAverage
				consensusClientResponseTimeDelayed = strconv.FormatUint(uint64(r.AttestationDelayCount), 10) + "/" + strconv.FormatUint(uint64(r.AttestationTimeCount), 10)
				isSet = true
			}
		}

		consensusReport.AddRecord(report.ConsensusRecord{
			ConsensusClientResponseTimeAvg:     consensusResponseTimeAvg,
			ConsensusClientResponseTimeDelayed: consensusClientResponseTimeDelayed,
		})

		operatorReport.Render()
		consensusReport.Render()

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
