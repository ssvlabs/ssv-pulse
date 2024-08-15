package infrastructure

import (
	"fmt"

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
}

func NewMemoryMetric(name string, healthCondition []metric.HealthCondition[uint64]) *MemoryMetric {
	return &MemoryMetric{
		Base: metric.Base[uint64]{
			HealthConditions: healthCondition,
			Name:             name,
		},
	}
}

func (m *MemoryMetric) Measure() {
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
		"total":  toMegabytes(memory.Total),
		"used":   toMegabytes(memory.Used),
		"cached": toMegabytes(memory.Cached),
		"free":   toMegabytes(memory.Free),
	})
}

func (p *MemoryMetric) AggregateResults() string {
	var values map[string][]float64 = make(map[string][]float64)

	for _, point := range p.DataPoints {
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
