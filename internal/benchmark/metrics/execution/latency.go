package execution

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
	// durationHistogram accumulates all samples for the whole run and backs
	// both the shutdown report and health evaluation. Samples are truncated
	// (toward zero) to millisecond precision before being observed, which
	// keeps the distinct-value count bounded by the value domain rather than
	// by runtime. Truncation is safe for health evaluation because the P90
	// threshold is a whole number of milliseconds (time.Second): truncating
	// can never move a sub-threshold value up across the threshold the way
	// rounding could (e.g. 999.6ms rounding to 1s), so the <1s vs >=1s
	// classification is preserved exactly.
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
		logger.WriteError(metric.ExecutionGroup, l.Name, err)
		return
	}
	defer conn.Close()

	latency = time.Since(start)

	l.durationHistogram.Observe(latency.Truncate(time.Millisecond))

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

	logger.WriteMetric(metric.ExecutionGroup, l.Name, map[string]any{
		DurationMinMeasurement: percentiles[0],
		DurationP10Measurement: percentiles[10],
		DurationP50Measurement: percentiles[50],
		DurationP90Measurement: percentiles[90],
		DurationMaxMeasurement: percentiles[100],
	})
}

func (l *LatencyMetric) AggregateResults() string {
	percentiles := l.durationHistogram.Percentiles(0, 10, 50, 90, 100)

	return metric.FormatPercentiles(
		percentiles[0],
		percentiles[10],
		percentiles[50],
		percentiles[90],
		percentiles[100])
}
