package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGivenRangeOfPercentilesWhenCalculatePercentilesThenCalculatesCorrectly(t *testing.T) {
	tests := []struct {
		name        string
		values      []int
		percentiles []float64
		expected    map[float64]int
	}{
		{
			name:        "Valid percentiles",
			values:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			percentiles: []float64{10, 50, 90},
			expected:    map[float64]int{10: 1, 50: 5, 90: 9},
		},
		{
			name:        "No percentiles provided",
			values:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			percentiles: []float64{},
			expected:    map[float64]int{},
		},
		{
			name:        "Empty values list",
			values:      []int{},
			percentiles: []float64{10, 50, 90},
			expected:    map[float64]int{10: 0, 50: 0, 90: 0},
		},
		{
			name:        "Empty values and no percentiles",
			values:      []int{},
			percentiles: []float64{},
			expected:    map[float64]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePercentiles(tt.values, tt.percentiles...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGivenObservationsWhenHistogramPercentilesThenMatchesCalculatePercentiles(t *testing.T) {
	tests := []struct {
		name        string
		values      []int
		percentiles []float64
	}{
		{
			name:        "Distinct values",
			values:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			percentiles: []float64{0, 10, 50, 90, 100},
		},
		{
			name:        "Repeated values",
			values:      []int{5, 5, 5, 1, 1, 9, 9, 9, 9, 3},
			percentiles: []float64{0, 10, 50, 90, 100},
		},
		{
			name:        "Single value repeated",
			values:      []int{7, 7, 7, 7},
			percentiles: []float64{0, 50, 100},
		},
		{
			name:        "No percentiles requested",
			values:      []int{1, 2, 3},
			percentiles: []float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHistogram[int]()
			for _, v := range tt.values {
				h.Observe(v)
			}

			expected := CalculatePercentiles(append([]int{}, tt.values...), tt.percentiles...)
			actual := h.Percentiles(tt.percentiles...)

			assert.Equal(t, expected, actual)
		})
	}
}

func TestGivenNoObservationsWhenHistogramPercentilesThenReturnsZeroValues(t *testing.T) {
	h := NewHistogram[int]()

	result := h.Percentiles(10, 50, 90)

	assert.Equal(t, map[float64]int{10: 0, 50: 0, 90: 0}, result)
}

func TestGivenFewerObservationsThanCapacityWhenRingBufferPercentilesThenExactOverWhatWasObserved(t *testing.T) {
	r := NewRingBuffer[int](5)
	for _, v := range []int{1, 2, 3} {
		r.Observe(v)
	}

	result := r.Percentiles(0, 50, 100)

	assert.Equal(t, map[float64]int{0: 1, 50: 2, 100: 3}, result)
}

func TestGivenMoreObservationsThanCapacityWhenRingBufferPercentilesThenOnlyMostRecentAreKept(t *testing.T) {
	r := NewRingBuffer[int](3)
	for _, v := range []int{100, 100, 100, 1, 2, 3} { // first three should be overwritten
		r.Observe(v)
	}

	result := r.Percentiles(0, 50, 100)

	assert.Equal(t, map[float64]int{0: 1, 50: 2, 100: 3}, result)
}

func TestGivenNoObservationsWhenRingBufferPercentilesThenReturnsZeroValues(t *testing.T) {
	r := NewRingBuffer[int](5)

	result := r.Percentiles(10, 50, 90)

	assert.Equal(t, map[float64]int{10: 0, 50: 0, 90: 0}, result)
}
