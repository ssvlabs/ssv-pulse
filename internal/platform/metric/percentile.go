package metric

import (
	"fmt"
	"slices"
	"sync"
)

type (
	stringable interface {
		~int | ~int64 | ~float64 | ~uint64 | ~uint16 | ~uint32 | string
	}
)

func CalculatePercentiles[T Metricable](values []T, percentiles ...float64) map[float64]T {
	result := make(map[float64]T)

	if len(percentiles) == 0 {
		return result
	}

	if len(values) == 0 {
		var zero T
		for _, p := range percentiles {
			result[p] = zero
		}
		return result
	}

	slices.Sort(values)

	for _, percentile := range percentiles {
		index := int(float64(len(values)-1) * percentile / 100.0)
		result[percentile] = values[index]
	}

	return result
}

func FormatPercentiles[T stringable](min, p10, p50, p90, max T) string {
	return fmt.Sprintf("min=%v, p10=%v, p50=%v, p90=%v, max=%v", min, p10, p50, p90, max)
}

// Histogram tracks a frequency count per distinct observed value, allowing
// exact percentiles to be computed over an unbounded number of observations
// with memory bounded by the number of distinct values rather than the
// number of observations. Callers of Observe should round/bucket continuous
// values (e.g. to a sane time or percentage precision) to keep the number of
// distinct values bounded by the value domain instead of by elapsed time.
type Histogram[T Metricable] struct {
	mu     sync.Mutex
	counts map[T]uint64
}

func NewHistogram[T Metricable]() *Histogram[T] {
	return &Histogram[T]{counts: make(map[T]uint64)}
}

func (h *Histogram[T]) Observe(value T) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.counts[value]++
}

// Percentiles returns, for each requested percentile, the value that would
// appear at that percentile's index if every observation were expanded into
// a flat sorted slice and indexed the same way CalculatePercentiles does.
func (h *Histogram[T]) Percentiles(percentiles ...float64) map[float64]T {
	result := make(map[float64]T)

	if len(percentiles) == 0 {
		return result
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var total uint64
	for _, count := range h.counts {
		total += count
	}

	if total == 0 {
		var zero T
		for _, p := range percentiles {
			result[p] = zero
		}
		return result
	}

	values := make([]T, 0, len(h.counts))
	for value := range h.counts {
		values = append(values, value)
	}
	slices.Sort(values)

	for _, percentile := range percentiles {
		targetIndex := uint64(float64(total-1) * percentile / 100.0)

		var cumulative uint64
		for _, value := range values {
			cumulative += h.counts[value]
			if cumulative > targetIndex {
				result[percentile] = value
				break
			}
		}
	}

	return result
}

// RingBuffer retains only the most recent N observations, giving exact
// (unrounded) percentiles over a bounded, constant-size recent window. Use
// this instead of Histogram when a caller needs an exact value — e.g. a
// live health-threshold check — rather than an approximation bucketed for
// whole-run reporting.
type RingBuffer[T Metricable] struct {
	mu     sync.Mutex
	values []T
	next   int
}

func NewRingBuffer[T Metricable](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{values: make([]T, 0, capacity)}
}

func (r *RingBuffer[T]) Observe(value T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.values) < cap(r.values) {
		r.values = append(r.values, value)
		return
	}

	r.values[r.next] = value
	r.next = (r.next + 1) % len(r.values)
}

// Percentiles returns exact percentiles over whatever is currently in the
// window (fewer than capacity observations early in a run).
func (r *RingBuffer[T]) Percentiles(percentiles ...float64) map[float64]T {
	r.mu.Lock()
	snapshot := make([]T, len(r.values))
	copy(snapshot, r.values)
	r.mu.Unlock()

	return CalculatePercentiles(snapshot, percentiles...)
}
