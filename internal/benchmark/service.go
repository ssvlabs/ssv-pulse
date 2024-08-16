package benchmark

import (
	"context"
	"log/slog"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/benchmark/report"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
	"golang.org/x/exp/maps"
)

type (
	metricService interface {
		Measure()
		GetName() string
		AggregateResults() string
		EvaluateMetric() (metric.HealthStatus, map[string]metric.SeverityLevel)
	}
	reportService interface {
		AddRecord(metric report.Record)
		Render()
	}

	Service struct {
		metrics map[metric.Group][]metricService
		report  reportService
	}
)

func New(
	metrics map[metric.Group][]metricService,
	reportService reportService,
) *Service {
	return &Service{
		metrics: metrics,
		report:  reportService,
	}
}

func (s *Service) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	var enabledMetrics map[metric.Group][]string = make(map[metric.Group][]string)
	for group, metrics := range s.metrics {
		for _, metric := range metrics {
			enabledMetrics[group] = append(enabledMetrics[group], metric.GetName())
		}
	}

	slog.
		With("groups", maps.Keys(enabledMetrics)).
		With("metrics", maps.Values(enabledMetrics)).
		Debug("starting benchmark service")

	for {
		select {
		case <-ticker.C:
			for _, groupMetrics := range s.metrics {
				for _, metric := range groupMetrics {
					go metric.Measure()
				}
			}
		case <-ctx.Done():
			for metricGroup, groupMetrics := range s.metrics {
				for _, m := range groupMetrics {
					health, severity := m.EvaluateMetric()

					slog.With("metric_group", metricGroup).With("metric_name", m.GetName()).Info("adding report record")
					s.report.AddRecord(report.Record{
						GroupName:  metricGroup,
						MetricName: m.GetName(),
						Value:      m.AggregateResults(),
						Health:     health,
						Severity:   severity,
					})
				}
			}

			slog.Info("rendering")
			s.report.Render()
			return
		}
	}
}
