package consensus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ssvlabs/ssv-pulse/internal/platform/metric"
)

// observeLatency mirrors what measure() does minus the TCP dial: bucket the
// sample into the histogram exactly as production does, then run the health
// bookkeeping in writeMetric.
func observeLatency(l *LatencyMetric, latency time.Duration) {
	l.durationHistogram.Observe(latency.Truncate(time.Millisecond))
	l.writeMetric(latency)
}

func newHealthEvaluatedLatencyMetric() *LatencyMetric {
	return NewLatencyMetric("host:5052", "Latency", 3*time.Second, []metric.HealthCondition[time.Duration]{
		{Name: DurationP90Measurement, Threshold: time.Second, Operator: metric.OperatorGreaterThanOrEqual, Severity: metric.SeverityHigh},
	})
}

// TestGivenLatencyNearOneSecondWhenEvaluateThenTruncationPreservesThreshold is
// the crux of using Truncate rather than Round: a sample just under 1s must
// stay healthy (Round would have pushed 999.6ms up to 1s and tripped the
// >=1s condition), while a sample at or above 1s must read unhealthy.
func TestGivenLatencyNearOneSecondWhenEvaluateThenTruncationPreservesThreshold(t *testing.T) {
	tests := []struct {
		name    string
		latency time.Duration
		want    metric.HealthStatus
	}{
		{name: "just under 1s stays healthy", latency: 999600 * time.Microsecond, want: metric.Healthy},
		{name: "exactly 1s is unhealthy", latency: time.Second, want: metric.Unhealthy},
		{name: "just over 1s is unhealthy", latency: 1000400 * time.Microsecond, want: metric.Unhealthy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newHealthEvaluatedLatencyMetric()

			observeLatency(l, tt.latency)

			health, _ := l.EvaluateMetric()
			assert.Equal(t, tt.want, health)
		})
	}
}

// TestGivenSpikeThenRecoveryWhenEvaluateThenWorstEverSeverityIsRetained
// confirms the cumulative contract: once any tick's P90 crosses the
// threshold the metric stays unhealthy, even though later healthy samples
// drag the reported P90 back under 1s.
func TestGivenSpikeThenRecoveryWhenEvaluateThenWorstEverSeverityIsRetained(t *testing.T) {
	l := newHealthEvaluatedLatencyMetric()

	observeLatency(l, 1200*time.Millisecond) // spike: P90 == 1.2s -> unhealthy
	for range 50 {
		observeLatency(l, 10*time.Millisecond) // recovery: cumulative P90 falls back under 1s
	}

	health, severities := l.EvaluateMetric()
	assert.Equal(t, metric.Unhealthy, health)
	assert.Equal(t, metric.SeverityHigh, severities[DurationP90Measurement])
}
