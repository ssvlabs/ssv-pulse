package loki

import (
	"bufio"
	"encoding/json"
	"errors"
	"github.com/grafana/loki-client-go/loki"
	"github.com/grafana/loki-client-go/pkg/urlutil"
	"github.com/prometheus/common/model"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	folderFlag   = "folder"
	lokiURLFlag  = "loki-url"
	usernameFlag = "username"
	passwordFlag = "password"
	labelsFlag   = "labels"
)

func init() {
	addFlags(CMD)
	if err := bindFlags(CMD); err != nil {
		panic(err.Error())
	}
}

var CMD = &cobra.Command{
	Use:   "send-loki",
	Short: "Send logs to Loki instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		if valid, err := Validate(); !valid {
			return err
		}
		return sendLogsToLoki(viper.GetString(folderFlag), viper.GetString(lokiURLFlag), viper.GetStringMapString(labelsFlag))
	},
}

func Validate() (bool, error) {
	if viper.GetString(folderFlag) == "" || viper.GetString(lokiURLFlag) == "" {
		return false, errors.New("folder and loki-url are required")
	}
	return true, nil
}

func sendLogsToLoki(folder, lokiURL string, labels map[string]string) error {
	var serverURL urlutil.URLValue
	if err := serverURL.Set(lokiURL); err != nil {
		slog.With("error", err).Error("Failed to set Loki URL")
		return err
	}
	cfg, err := loki.NewDefaultConfig(lokiURL)
	if err != nil {
		slog.With("error", err).Error("Failed to create default Loki config")
		return err
	}

	client, err := loki.New(cfg)
	if err != nil {
		slog.With("error", err).Error("Failed to create Loki client")
		return err
	}
	defer func() {
		slog.Info("Stopping Loki client, waiting for pending logs to be sent...")
		client.Stop()
	}()

	labelSet := model.LabelSet{"job": model.LabelValue("ssv-loki")}
	for key, value := range labels {
		labelSet[model.LabelName(key)] = model.LabelValue(value)
	}

	var wg sync.WaitGroup
	fileChan := make(chan string, 10)

	go func() {
		if err := filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				slog.With("path", path, "error", err).Warn("Failed to access path")
				return err
			}
			if !d.IsDir() && filepath.Ext(d.Name()) == ".log" {
				fileChan <- path
			}
			return nil
		}); err != nil {
			slog.With("error", err).Error("Error walking the path")
		}
		close(fileChan)
	}()

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				currentLs := labelSet.Clone()
				currentLs["filename"] = model.LabelValue(filepath.Base(path))
				if err := processFile(client, path, currentLs); err != nil {
					slog.With("path", path, "error", err).Warn("Failed to process file")
				}
			}
		}()
	}

	wg.Wait()
	return nil
}

func processFile(client *loki.Client, filePath string, labelSet model.LabelSet) error {
	logFile, err := os.Open(filePath)
	if err != nil {
		slog.With("filePath", filePath, "error", err).Warn("Failed to open file")
		return err
	}
	defer logFile.Close()

	scanner := bufio.NewScanner(logFile)
	slog.With("filename", filePath).With("labels", labelSet).Info("Sending logs to Loki")
	for scanner.Scan() {
		line := scanner.Text()

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			slog.With("error", err).Warn("Failed to unmarshal JSON")
			continue
		}

		if timestampStr, ok := entry["T"].(string); ok {
			timestamp, err := time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				slog.With("error", err).Warn("Failed to parse timestamp")
				continue
			}

			if err := client.Handle(labelSet, timestamp, line); err != nil {
				slog.With("error", err).Warn("Failed to send log to Loki")
			}
		} else {
			slog.With("entry", entry).Warn("Log entry missing 'T' field")
		}
	}

	if err := scanner.Err(); err != nil {
		slog.With("error", err).Warn("Failed to process file")
	}
	return nil
}

func addFlags(cobraCMD *cobra.Command) {
	cobraCMD.Flags().String(folderFlag, "", "Folder containing JSON log files")
	cobraCMD.Flags().String(lokiURLFlag, "", "URL of the Loki instance")
	cobraCMD.Flags().String(usernameFlag, "", "Username for Loki authentication (optional)")
	cobraCMD.Flags().String(passwordFlag, "", "Password for Loki authentication (optional)")
	cobraCMD.Flags().StringToString(labelsFlag, map[string]string{}, "Labels to add to logs (optional)")
}

func bindFlags(cmd *cobra.Command) error {
	if err := viper.BindPFlag(folderFlag, cmd.Flags().Lookup(folderFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag(lokiURLFlag, cmd.Flags().Lookup(lokiURLFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag(usernameFlag, cmd.Flags().Lookup(usernameFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag(passwordFlag, cmd.Flags().Lookup(passwordFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag(labelsFlag, cmd.Flags().Lookup(labelsFlag)); err != nil {
		return err
	}
	return nil
}
