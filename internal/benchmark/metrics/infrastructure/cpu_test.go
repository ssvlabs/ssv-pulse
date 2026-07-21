package infrastructure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGivenSubPercentPrecisionSamplesWhenAggregateResultsThenRoundsToTwoDecimals(t *testing.T) {
	c := NewCPUMetric("CPU", time.Second, nil)

	// system 12.345 -> 12.35 (round half away from zero), user 67.891 -> 67.89.
	c.writeMetric(12.345, 67.891)

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
