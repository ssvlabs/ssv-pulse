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
		filesJson := make([]os.DirEntry, 0, len(filesAll))
		for _, file := range filesAll {
			if file.IsDir() {
				continue
			}
			if strings.HasSuffix(file.Name(), ".log") || strings.HasSuffix(file.Name(), ".txt") {
				filesLog = append(filesLog, file)
			}
			if strings.HasSuffix(file.Name(), ".json") {
				filesJson = append(filesJson, file)
			}
		}

		if len(filesLog) == 0 && len(filesJson) == 0 {
			return fmt.Errorf("no log files found in %s", cfg.LogFilesDirectory)
		}

		blockchain, err := environment.BlockchainByName(cfg.Blockchain)
		if err != nil {
			return fmt.Errorf("get blockchain by name: %w", err)
		}

		logParser, err := environment.LogParserByName(cfg.LogParser)
		if err != nil {
			return fmt.Errorf("get log parser by name: %w", err)
		}

		if cfg.AnalyzeCommitteeDuty {
			slog.Info(fmt.Sprintf("analyzing COMMITTEE duty for target slot %d", cfg.TargetSlot))
			a := duties.NewCommittee(blockchain, logParser)
			err := duties.Analyze(a, cfg.LogFilesDirectory, filesLog, cfg.TargetSlot)
			if err != nil {
				return fmt.Errorf("analyze proposer duty: %w", err)
			}
		}
		if cfg.AnalyzeProposerDuty {
			slog.Info(fmt.Sprintf("analyzing PROPOSER duty for target slot %d", cfg.TargetSlot))
			a := duties.NewProposer(blockchain, logParser)
			err := duties.Analyze(a, cfg.LogFilesDirectory, filesLog, cfg.TargetSlot)
			if err != nil {
				return fmt.Errorf("analyze proposer duty: %w", err)
			}
		}

		// TODO - need this ?
		//for _, file := range filesJson {
		//	filePath := path.Join(cfg.LogFilesDirectory, file.Name())
		//
		//	fileSizeMB := 0.0
		//	stat, err := os.Stat(filePath)
		//	if err != nil {
		//		slog.With("err", err.Error()).Warn(fmt.Sprintf("error fetching `%s` file info, will try to read the file anyway", file.Name()))
		//	}
		//	if err == nil {
		//		fileSizeMB = float64(stat.Size()) / (1024 * 1024)
		//	}
		//	slog.
		//		With("file_size_megabytes", math.Round(fileSizeMB)).
		//		Info(fmt.Sprintf("⏳⏳⏳ analyzing json file: %s", file.Name()))
		//
		//	// TODO
		//	//err = proposer.AnalyzeJson(filePath)
		//	//if err != nil {
		//	//	return fmt.Errorf("proposer: analyze file: %w", err)
		//	//}
		//
		//	err = commitee.AnalyzeJson(filePath)
		//	if err != nil {
		//		return fmt.Errorf("commitee: analyze file: %w", err)
		//	}
		//}

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
