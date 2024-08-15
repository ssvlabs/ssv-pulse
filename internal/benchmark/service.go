package benchmark

import (
	"context"
	"log/slog"

	"github.com/ssvlabs/ssv-benchmark/internal/benchmark/report"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

type (
	metricService interface {
		Start(context.Context) (map[string]metric.GroupResult, error)
	}

	reportService interface {
		AddRecord(metric report.Record)
		Render()
	}

	Service struct {
		metrics map[metric.Group]metricService
		report  reportService
	}
)

func New(
	metrics map[metric.Group]metricService,
	reportService reportService,
) *Service {
	return &Service{
		metrics: metrics,
		report:  reportService,
	}
}

func (s *Service) Start(ctx context.Context) {
	metricCount := len(s.metrics)
	var writtenMetrics uint8
	for metricGroup, metricSvc := range s.metrics {
		slog.With("group", metricGroup).Info("launching metric service")

		go func(ctx context.Context) {
			result, err := metricSvc.Start(ctx)
			if err == context.DeadlineExceeded || err == context.Canceled {
				slog.With("err", err.Error()).With("group", metricGroup).Warn("service was shut down")
			}

			for metricName, metricResult := range result {
				slog.With("metric_group", metricGroup).With("metric_name", metricName).Info("adding report record")
				s.report.AddRecord(report.Record{
					GroupName:  metricGroup,
					MetricName: metricName,
					Value:      metricResult.ViewResult,
					Health:     metricResult.Health,
					Severity:   metricResult.Severity,
				})
			}

			writtenMetrics++
			if writtenMetrics == uint8(metricCount) {
				slog.Info("rendering")
				s.report.Render()
			}
		}(ctx)
	}
}
