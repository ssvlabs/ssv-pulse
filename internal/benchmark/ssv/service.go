package ssv

import (
	"context"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

type Service struct {
	interval time.Duration
	metrics  []metric.Metric
}

func New(metrics []metric.Metric) *Service {
	return &Service{
		metrics:  metrics,	
		interval: time.Second * 5,
	}
}

func (s *Service) Start(ctx context.Context) (map[string]metric.GroupResult, error) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, metric := range s.metrics {
				metric.Measure()
			}
		case <-ctx.Done():
			var result map[string]metric.GroupResult = make(map[string]metric.GroupResult, len(s.metrics))

			for _, m := range s.metrics {
				health, severity := m.EvaluateMetric()

				result[m.GetName()] = metric.GroupResult{
					ViewResult: m.AggregateResults(),
					Health:     health,
					Severity:   severity,
				}
			}
			return result, ctx.Err()
		}
	}
}
