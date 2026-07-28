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

// truncateGranularity buckets samples before they enter the histogram, keeping
// its distinct-value count bounded by the value domain instead of by runtime
// (~22.5k keys at most, given the dial timeout). It is fine-grained enough that
// sub-millisecond dials — typical for local or port-forwarded nodes — still
// report real values rather than flooring to zero.
//
// Every P90 health threshold must be a whole multiple of this: truncation
// rounds toward zero, so that guarantees truncate(v) >= threshold exactly when
// v >= threshold, preserving the health classification (unlike rounding, which
// could push e.g. 999.6ms up to 1s).
const truncateGranularity = 100 * time.Microsecond

type LatencyMetric struct {
	metric.Base[time.Duration]
	host              string
	interval, timeout time.Duration
	// Accumulates all samples for the whole run, backing both the shutdown
	// report and health evaluation. See truncateGranularity.
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

	l.durationHistogram.Observe(latency.Truncate(truncateGranularity))

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
