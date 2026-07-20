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

// recentSampleWindow bounds the ring buffer that feeds health evaluation to
// the most recent ~5 minutes of samples at the default 3s interval — enough
// to give a meaningful P90, small enough to stay exact and O(1) in memory.
//
// This is a deliberate change from the original behavior, which evaluated
// health against a cumulative whole-run P90 recomputed at every tick: a
// short-lived latency spike can now trip the health condition even if the
// whole-run average would look fine. This is intentional — a rolling recent
// window is a more standard, more responsive health signal than an
// ever-diluting lifetime average — but it is a real behavioral change, not
// just an implementation detail. Health itself is still only ever read
// once, at shutdown (see Service.Start), not polled live during the run;
// what changed is what that one shutdown-time judgment is based on.
const recentSampleWindow = 100

type LatencyMetric struct {
	metric.Base[time.Duration]
	host              string
	interval, timeout time.Duration
	// durationHistogram backs AggregateResults' whole-run report and buckets
	// samples to millisecond precision to stay memory-bounded; recentWindow
	// backs health evaluation with exact, unrounded recent samples so
	// bucketing can never flip a threshold classification (see
	// recentSampleWindow for why these two intentionally differ).
	durationHistogram *metric.Histogram[time.Duration]
	recentWindow      *metric.RingBuffer[time.Duration]
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
		recentWindow:      metric.NewRingBuffer[time.Duration](recentSampleWindow),
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

	l.durationHistogram.Observe(latency.Round(time.Millisecond))
	l.recentWindow.Observe(latency)

	l.writeMetric(latency)
}

func (l *LatencyMetric) writeMetric(latency time.Duration) {
	// Exact, unrounded percentiles over the recent window — this feeds
	// AddDataPoint below, which incrementally updates the health state that
	// EvaluateMetric reads once at shutdown, so bucketing can't shift a
	// value across the health threshold.
	percentiles := l.recentWindow.Percentiles(0, 10, 50, 90, 100)

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
