package benchmark

import (
	"context"
	"log/slog"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

type (
	MetricService interface {
		Start(context.Context) error
	}

	Service struct {
		metrics map[metric.Group]MetricService
	}
)

func New(metrics map[metric.Group]MetricService) *Service {
	return &Service{
		metrics: metrics,
	}
}

func (s *Service) Start(ctx context.Context) {
	for metricGroup, metricSvc := range s.metrics {
		slog.With("group", metricGroup).Info("launching metric service")

		go func(ctx context.Context) {
			if err := metricSvc.Start(ctx); err == context.DeadlineExceeded {
				slog.With("group", metricGroup).Info("service was shut down")
			}
		}(ctx)
	}
}
