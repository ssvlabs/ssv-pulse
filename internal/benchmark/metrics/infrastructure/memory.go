package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mackerelio/go-osstat/memory"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	UsedMemoryMeasurement   = "Used"
	TotalMemoryMeasurement  = "Total"
	CachedMemoryMeasurement = "Cached"
	FreeMemoryMeasurement   = "Free"
)

type MemoryMetric struct {
	metric.Base[uint64]
	interval time.Duration
}

func NewMemoryMetric(name string, interval time.Duration, healthCondition []metric.HealthCondition[uint64]) *MemoryMetric {
	return &MemoryMetric{
		Base: metric.Base[uint64]{
			HealthConditions: healthCondition,
			Name:             name,
		},
		interval: interval,
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

	m.AddDataPoint(map[string]uint64{
		CachedMemoryMeasurement: memory.Cached,
		UsedMemoryMeasurement:   memory.Used,
		FreeMemoryMeasurement:   memory.Free,
		TotalMemoryMeasurement:  memory.Total,
	})

	logger.WriteMetric(metric.InfrastructureGroup, m.Name, map[string]any{
		TotalMemoryMeasurement:  toMegabytes(memory.Total),
		UsedMemoryMeasurement:   toMegabytes(memory.Used),
		CachedMemoryMeasurement: toMegabytes(memory.Cached),
		FreeMemoryMeasurement:   toMegabytes(memory.Free),
	})
}

func (m *MemoryMetric) AggregateResults() string {
	var values map[string][]float64 = make(map[string][]float64)

	for _, point := range m.DataPoints {
		values[TotalMemoryMeasurement] = append(values[TotalMemoryMeasurement], toMegabytes(point.Values[TotalMemoryMeasurement]))
		values[FreeMemoryMeasurement] = append(values[FreeMemoryMeasurement], toMegabytes(point.Values[FreeMemoryMeasurement]))
		values[UsedMemoryMeasurement] = append(values[UsedMemoryMeasurement], toMegabytes(point.Values[UsedMemoryMeasurement]))
		values[CachedMemoryMeasurement] = append(values[CachedMemoryMeasurement], toMegabytes(point.Values[CachedMemoryMeasurement]))
	}

	return fmt.Sprintf("total_P50=%.2fMB, used_P50=%.2fMB, cached_P50=%.2fMB, free_P50=%.2fMB",
		metric.CalculatePercentile(values[TotalMemoryMeasurement], 50),
		metric.CalculatePercentile(values[UsedMemoryMeasurement], 50),
		metric.CalculatePercentile(values[CachedMemoryMeasurement], 50),
		metric.CalculatePercentile(values[FreeMemoryMeasurement], 50))
}

func toMegabytes(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024)
}
