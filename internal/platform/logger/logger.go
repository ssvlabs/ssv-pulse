package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/metric"
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	slog.SetDefault(logger)
	slog.Debug("logger initialized")
}

func WriteMetric(metricGroup metric.Group, metricName metric.Name, nameValue map[string]any) {
	logger := slog.Default()

	logger = logger.With("values", nameValue)

	logger.
		With("metric_group", strings.ToLower(string(metricGroup))).
		With("metric_name", strings.ToLower(string(metricName))).
		Info("measured")
}

func WriteError(metricGroup metric.Group, metricName metric.Name, err error) {
	slog.
		With("metric_group", strings.ToLower(string(metricGroup))).
		With("metric_name", strings.ToLower(string(metricName))).
		Info("error")
}
