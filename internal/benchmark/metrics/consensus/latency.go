package consensus

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/platform/logger"
	"github.com/ssvlabs/ssv-pulse/internal/platform/metric"
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
	host              string
	interval, timeout time.Duration
	durationHistogram *metric.Histogram[time.Duration]
}

func NewLatencyMetric(host, name string, interval time.Duration, healthCondition []metric.HealthCondition[time.Duration]) *LatencyMetric {
	return &LatencyMetric{
		host: host,
		Base: metric.Base[time.Duration]{
			HealthConditions: healthCondition,
			Name:             name,
		},
		interval:          interval,
		timeout:           time.Duration(float64(interval) * 0.75),
		durationHistogram: metric.NewHistogram[time.Duration](),
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
			l.measure()
		}
	}
}

func (l *LatencyMetric) measure() {
	var latency time.Duration
	start := time.Now()

	conn, err := net.DialTimeout("tcp", l.host, l.timeout)
	if err != nil {
		logger.WriteError(metric.ConsensusGroup, l.Name, err)
		return
	}
	defer conn.Close()

	latency = time.Since(start)

	l.durationHistogram.Observe(latency.Round(time.Millisecond))

	l.writeMetric(latency)
}

func (l *LatencyMetric) writeMetric(latency time.Duration) {
	percentiles := l.durationHistogram.Percentiles(0, 10, 50, 90, 100)

	l.AddDataPoint(map[string]time.Duration{
		DurationMinMeasurement: percentiles[0],
		DurationP10Measurement: percentiles[10],
		DurationP50Measurement: percentiles[50],
		DurationP90Measurement: percentiles[90],
		DurationMaxMeasurement: percentiles[100],
	})

	latencyMetric.With(serverAddrLabel(l.host)).Observe(latency.Seconds())

	logger.WriteMetric(metric.ConsensusGroup, l.Name, map[string]any{
		DurationMinMeasurement: percentiles[0],
		DurationP10Measurement: percentiles[10],
		DurationP50Measurement: percentiles[50],
		DurationP90Measurement: percentiles[90],
		DurationMaxMeasurement: percentiles[100],
	})
}

func (l *LatencyMetric) AggregateResults() string {
	min := l.LastValue(DurationMinMeasurement)
	p10 := l.LastValue(DurationP10Measurement)
	p50 := l.LastValue(DurationP50Measurement)
	p90 := l.LastValue(DurationP90Measurement)
	max := l.LastValue(DurationMaxMeasurement)

	return metric.FormatPercentiles(min, p10, p50, p90, max)
}
