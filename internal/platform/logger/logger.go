package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	slog.Debug("logger initialized")
}

func WriteMetric(metricGroup metric.Group, metricName string, nameValue map[string]any) {
	logger := slog.Default()

	logger = logger.With("values", nameValue)

	logger.
		With("metric_group", strings.ToLower(string(metricGroup))).
		With("metric_name", strings.ToLower(string(metricName))).
		Debug("measured")
}

func WriteError(metricGroup metric.Group, metricName string, err error) {
	slog.
		With("err", err.Error()).
		With("metric_group", strings.ToLower(string(metricGroup))).
		With("metric_name", strings.ToLower(string(metricName))).
		Error("error")
}
