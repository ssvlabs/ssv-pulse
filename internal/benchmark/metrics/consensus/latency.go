package consensus

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ssvlabsinfra/ssv-pulse/internal/platform/logger"
	"github.com/ssvlabsinfra/ssv-pulse/internal/platform/metric"
)

const (
	DurationMeasurement = "Duration"
)

type LatencyMetric struct {
	metric.Base[time.Duration]
	url      string
	interval time.Duration
}

func NewLatencyMetric(url, name string, interval time.Duration, healthCondition []metric.HealthCondition[time.Duration]) *LatencyMetric {
	return &LatencyMetric{
		url: url,
		Base: metric.Base[time.Duration]{
			HealthConditions: healthCondition,
			Name:             name,
		},
		interval: interval,
	}
}

func (l *LatencyMetric) Measure(ctx context.Context) {
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.With("metric_name", l.Name).Debug("metric was stopped")
			return
		case <-ticker.C:
			l.measure(ctx)
		}
	}
}

func (l *LatencyMetric) measure(ctx context.Context) {
	var latency time.Duration
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.url, nil)
	if err != nil {
		logger.WriteError(metric.ConsensusGroup, l.Name, err)
		return
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.WriteError(metric.ConsensusGroup, l.Name, err)
		return
	}
	defer res.Body.Close()

	end := time.Now()

	latency = end.Sub(start)

	l.AddDataPoint(map[string]time.Duration{
		DurationMeasurement: latency,
	})

	logger.WriteMetric(metric.ConsensusGroup, l.Name, map[string]any{DurationMeasurement: latency})
}

func (l *LatencyMetric) AggregateResults() string {
	var values []time.Duration
	for _, point := range l.DataPoints {
		values = append(values, point.Values[DurationMeasurement])
	}
	return metric.FormatPercentiles(
		metric.CalculatePercentile(values, 0),
		metric.CalculatePercentile(values, 10),
		metric.CalculatePercentile(values, 50),
		metric.CalculatePercentile(values, 90),
		metric.CalculatePercentile(values, 100))
}
