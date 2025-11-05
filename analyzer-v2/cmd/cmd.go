package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabs/ssv-pulse/analyzer-v2/config"
	"github.com/ssvlabs/ssv-pulse/analyzer-v2/duties"
	"github.com/ssvlabs/ssv-pulse/analyzer-v2/internal/environment"
)

const (
	logFilesDirectoryFlag = "log-files-directory"
)

func init() {
	CMD.Flags().String(logFilesDirectoryFlag, "", "Path to the directory containing SSV node log files for analysis, e.g. my-file-dir")

	err := viper.BindPFlag("analyzer.log-files-directory", CMD.Flags().Lookup(logFilesDirectoryFlag))
	if err != nil {
		// TODO
		panic(err)
	}
}

func main() {
	err := CMD.Execute()
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

var CMD = &cobra.Command{
	Use:   "ssv-pulse-analyze-v2",
	Short: "CLI for analyzing log files emitted by SSV node",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := parseConfig()
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}

		slog.With("config", cfg).
			With("config_file", viper.ConfigFileUsed()).
			Info("configuration is loaded")

		err = cfg.Validate()
		if err != nil {
			return fmt.Errorf("validate config: %w", err)
		}

		filesAll, err := os.ReadDir(cfg.LogFilesDirectory)
		if err != nil {
			return fmt.Errorf("could not read logs directory: %w", err)
		}
		filesLog := make([]os.DirEntry, 0, len(filesAll))
		for _, file := range filesAll {
			if file.IsDir() {
				continue
			}
			if strings.HasSuffix(file.Name(), ".log") || strings.HasSuffix(file.Name(), ".logs") || strings.HasSuffix(file.Name(), ".txt") {
				filesLog = append(filesLog, file)
			}
		}

		if len(filesLog) == 0 {
			return fmt.Errorf("no log files found in %s", cfg.LogFilesDirectory)
		}

		blockchain, err := environment.BlockchainByName(cfg.Blockchain)
		if err != nil {
			return fmt.Errorf("get blockchain by name: %w", err)
		}

		logParser, err := environment.LogParserByName(cfg.LogFormat)
		if err != nil {
			return fmt.Errorf("get log parser by name: %w", err)
		}

		if cfg.AnalyzeCommitteeDuty {
			slog.Info(fmt.Sprintf("analyzing COMMITTEE duty: target slot=%d, duty_id=%s", cfg.TargetSlot, cfg.DutyID))
			a := duties.NewCommittee(blockchain, logParser)
			err := duties.Analyze(a, cfg.LogFilesDirectory, filesLog, cfg.DutyID, cfg.TargetSlot)
			if err != nil {
				return fmt.Errorf("analyze proposer duty: %w", err)
			}
		}
		if cfg.AnalyzeProposerDuty {
			slog.Info(fmt.Sprintf("analyzing PROPOSER duty: target slot=%d, duty_id=%s", cfg.TargetSlot, cfg.DutyID))
			a := duties.NewProposer(blockchain, logParser)
			err := duties.Analyze(a, cfg.LogFilesDirectory, filesLog, cfg.DutyID, cfg.TargetSlot)
			if err != nil {
				return fmt.Errorf("analyze proposer duty: %w", err)
			}
		}
		if cfg.AnalyzeAggregatorDuty {
			slog.Info(fmt.Sprintf("analyzing AGGREGATOR duty: target slot=%d, duty_id=%s", cfg.TargetSlot, cfg.DutyID))
			a := duties.NewAggregator(blockchain, logParser)
			err := duties.Analyze(a, cfg.LogFilesDirectory, filesLog, cfg.DutyID, cfg.TargetSlot)
			if err != nil {
				return fmt.Errorf("analyze aggregator duty: %w", err)
			}
		}
		if cfg.AnalyzeSyncCommitteeContribution {
			slog.Info(fmt.Sprintf("analyzing SYNC_COMMITTEE_CONTRIBUTION duty: target slot=%d, duty_id=%s", cfg.TargetSlot, cfg.DutyID))
			a := duties.NewSyncCommitteeContribution(blockchain, logParser)
			err := duties.Analyze(a, cfg.LogFilesDirectory, filesLog, cfg.DutyID, cfg.TargetSlot)
			if err != nil {
				return fmt.Errorf("analyze sync_committee_contribution duty: %w", err)
			}
		}

		return nil
	},
}

func parseConfig() (*config.Config, error) {
	viper.SetConfigName("config.yaml")
	viper.SetConfigType("yaml")

	viper.AddConfigPath(".")
	viper.AddConfigPath("./analyzer-v2/config")

	var cfg config.Config

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
