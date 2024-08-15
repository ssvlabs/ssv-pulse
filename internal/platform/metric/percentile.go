package metric

import (
	"fmt"
	"sort"

	"golang.org/x/exp/constraints"
)

type (
	Metricable interface {
		constraints.Ordered
	}

	stringable interface {
		~int | ~int64 | ~float64 | ~uint64 | ~uint16 | ~uint32 | string
	}
)

func CalculatePercentile[T Metricable](values []T, percentile float64) T {
	if len(values) == 0 {
		var zero T
		return zero
	}

	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})

	index := int(float64(len(values)-1) * percentile / 100.0)

	return values[index]
}

func FormatPercentiles[T stringable](min, p10, p50, p90, max T) string {
	return fmt.Sprintf("min=%v, p10=%v, p50=%v, p90=%v, max=%v", min, p10, p50, p90, max)
}
