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
	DurationMinMeasurement = "DurationMin"
	DurationP10Measurement = "DurationP10"
	DurationP50Measurement = "DurationP50"
	DurationP90Measurement = "DurationP90"
	DurationMaxMeasurement = "DurationMax"
)

type LatencyMetric struct {
	metric.Base[time.Duration]
	url       string
	interval  time.Duration
	durations []time.Duration
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

	l.durations = append(l.durations, latency)

	percentiles := metric.CalculatePercentiles(l.durations, 0, 10, 50, 90, 100)

	l.AddDataPoint(map[string]time.Duration{
		DurationMinMeasurement: percentiles[0],
		DurationP10Measurement: percentiles[10],
		DurationP50Measurement: percentiles[50],
		DurationP90Measurement: percentiles[90],
		DurationMaxMeasurement: percentiles[100],
	})

	logger.WriteMetric(metric.ConsensusGroup, l.Name, map[string]any{
		DurationMinMeasurement: percentiles[0],
		DurationP10Measurement: percentiles[10],
		DurationP50Measurement: percentiles[50],
		DurationP90Measurement: percentiles[90],
		DurationMaxMeasurement: percentiles[100],
	})
}

func (l *LatencyMetric) AggregateResults() string {
	var min, p10, p50, p90, max time.Duration

	min = l.DataPoints[len(l.DataPoints)-1].Values[DurationMinMeasurement]
	p10 = l.DataPoints[len(l.DataPoints)-1].Values[DurationP10Measurement]
	p50 = l.DataPoints[len(l.DataPoints)-1].Values[DurationP50Measurement]
	p90 = l.DataPoints[len(l.DataPoints)-1].Values[DurationP90Measurement]
	max = l.DataPoints[len(l.DataPoints)-1].Values[DurationMaxMeasurement]

	return metric.FormatPercentiles(min, p10, p50, p90, max)
}
