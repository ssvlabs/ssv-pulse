package benchmark

import (
	"context"
	"log/slog"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

type (
	MetricService interface {
		Start(context.Context)
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
		go metricSvc.Start(ctx)
	}
}
