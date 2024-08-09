package metric

import "sort"

type Numeric interface {
	~int | ~int64 | ~float64
}

func CalculatePercentile[T Numeric](values []T, percentile float64) T {
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
