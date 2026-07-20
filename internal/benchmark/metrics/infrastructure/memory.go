package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/mackerelio/go-osstat/memory"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/ssvlabs/ssv-pulse/internal/platform/logger"
	"github.com/ssvlabs/ssv-pulse/internal/platform/metric"
)

const (
	UsedMemoryMeasurement   = "Used"
	TotalMemoryMeasurement  = "Total"
	CachedMemoryMeasurement = "Cached"
	FreeMemoryMeasurement   = "Free"
)

type MemoryMetric struct {
	metric.Base[uint64]
	interval        time.Duration
	totalHistogram  *metric.Histogram[float64]
	usedHistogram   *metric.Histogram[float64]
	cachedHistogram *metric.Histogram[float64]
	freeHistogram   *metric.Histogram[float64]
}

func NewMemoryMetric(name string, interval time.Duration, healthCondition []metric.HealthCondition[uint64]) *MemoryMetric {
	return &MemoryMetric{
		Base: metric.Base[uint64]{
			HealthConditions: healthCondition,
			Name:             name,
		},
		interval:        interval,
		totalHistogram:  metric.NewHistogram[float64](),
		usedHistogram:   metric.NewHistogram[float64](),
		cachedHistogram: metric.NewHistogram[float64](),
		freeHistogram:   metric.NewHistogram[float64](),
	}
}

func (m *MemoryMetric) Measure(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.With("metric_name", m.Name).Debug("metric was stopped")
			return
		case <-ticker.C:
			m.measure()
		}
	}
}

func (m *MemoryMetric) measure() {
	memory, err := memory.Get()
	if err != nil {
		logger.WriteError(metric.InfrastructureGroup, m.Name, err)
		return
	}

	m.writeMetric(memory.Cached, memory.Used, memory.Free, memory.Total)
}

func (m *MemoryMetric) writeMetric(cached, used, free, total uint64) {
	// Round to the nearest whole MB before observing so the histogram's
	// cardinality stays bounded by system RAM size instead of growing with
	// every byte-level fluctuation between polls.
	m.totalHistogram.Observe(math.Round(toMegabytes(total)))
	m.usedHistogram.Observe(math.Round(toMegabytes(used)))
	m.cachedHistogram.Observe(math.Round(toMegabytes(cached)))
	m.freeHistogram.Observe(math.Round(toMegabytes(free)))

	m.AddDataPoint(map[string]uint64{
		CachedMemoryMeasurement: cached,
		UsedMemoryMeasurement:   used,
		FreeMemoryMeasurement:   free,
		TotalMemoryMeasurement:  total,
	})

	memoryUsageMetric.With(prometheus.Labels{memoryUsageTypeLabel: "cached"}).Set(float64(cached))
	memoryUsageMetric.With(prometheus.Labels{memoryUsageTypeLabel: "used"}).Set(float64(used))
	memoryUsageMetric.With(prometheus.Labels{memoryUsageTypeLabel: "free"}).Set(float64(free))
	memoryUsageMetric.With(prometheus.Labels{memoryUsageTypeLabel: "total"}).Set(float64(total))

	logger.WriteMetric(metric.InfrastructureGroup, m.Name, map[string]any{
		TotalMemoryMeasurement:  toMegabytes(total),
		UsedMemoryMeasurement:   toMegabytes(used),
		CachedMemoryMeasurement: toMegabytes(cached),
		FreeMemoryMeasurement:   toMegabytes(free),
	})
}

func (m *MemoryMetric) AggregateResults() string {
	return fmt.Sprintf("total_P50=%.2fMB, used_P50=%.2fMB, cached_P50=%.2fMB, free_P50=%.2fMB",
		m.totalHistogram.Percentiles(50)[50],
		m.usedHistogram.Percentiles(50)[50],
		m.cachedHistogram.Percentiles(50)[50],
		m.freeHistogram.Percentiles(50)[50])
}

func toMegabytes(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024)
}
