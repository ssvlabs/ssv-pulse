package infrastructure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const mb = 1024 * 1024

func TestGivenFractionalMegabyteSamplesWhenAggregateResultsThenReportsWholeMegabytes(t *testing.T) {
	m := NewMemoryMetric("Memory", time.Second, nil)

	// 100.68 MB total: rounds to 101. The report must print whole MB
	// (%.0f), not imply decimal precision the rounded histogram no longer
	// carries.
	m.writeMetric(50*mb, 150*mb, 60*mb, 100*mb+700*1024)

	result := m.AggregateResults()

	assert.Equal(t, "total_P50=101MB, used_P50=150MB, cached_P50=50MB, free_P50=60MB", result)
}
