package infrastructure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGivenSubPercentPrecisionSamplesWhenAggregateResultsThenRoundsToTwoDecimals(t *testing.T) {
	c := NewCPUMetric("CPU", time.Second, nil)

	// Inputs chosen away from X.XX5 half-way ties (whose float64
	// representation is not exact), so the rounding direction is
	// unambiguous: system 12.348 -> 12.35, user 67.892 -> 67.89.
	c.writeMetric(12.348, 67.892)

	result := c.AggregateResults()

	assert.Contains(t, result, "system_P50=12.35%")
	assert.Contains(t, result, "user_P50=67.89%")
}

func TestGivenRepeatedRoundingCollisionsWhenAggregateResultsThenBucketsShareOneKey(t *testing.T) {
	c := NewCPUMetric("CPU", time.Second, nil)

	// Values that round to the same 2-decimal bucket must collapse together,
	// keeping the histogram's cardinality bounded by the value domain.
	c.writeMetric(10.001, 20.002)
	c.writeMetric(10.004, 20.003)

	result := c.AggregateResults()

	assert.Contains(t, result, "system_P50=10.00%")
	assert.Contains(t, result, "user_P50=20.00%")
}
