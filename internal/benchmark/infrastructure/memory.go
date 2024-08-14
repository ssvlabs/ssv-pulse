package infrastructure

import (
	"errors"

	"github.com/mackerelio/go-osstat/memory"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

type MemoryMetric struct {
	totalMbs, usedMbs, cachedMbs, freeMbs []float64
}

func NewMemoryMetric() *MemoryMetric {
	return &MemoryMetric{}
}

func (m *MemoryMetric) Get() (totalMb, usedMb, cachedMb, freeMb float64, err error) {
	memory, err := memory.Get()
	if err != nil {
		return totalMb, usedMb, cachedMb, freeMb, errors.Join(err, errors.New("failed to measure memory metric"))
	}
	totalMb = toMegabytes(memory.Total)
	m.totalMbs = append(m.totalMbs, totalMb)

	usedMb = toMegabytes(memory.Used)
	m.usedMbs = append(m.usedMbs, usedMb)

	cachedMb = toMegabytes(memory.Cached)
	m.cachedMbs = append(m.cachedMbs, cachedMb)

	freeMb = toMegabytes(memory.Free)
	m.freeMbs = append(m.freeMbs, freeMb)

	return totalMb, usedMb, cachedMb, freeMb, err
}

func (c *MemoryMetric) Aggregate() (totalP50, usedP50, cachedP50, freeP50 float64) {
	totalP50 = metric.CalculatePercentile(c.totalMbs, 50)
	usedP50 = metric.CalculatePercentile(c.usedMbs, 50)
	cachedP50 = metric.CalculatePercentile(c.cachedMbs, 50)
	freeP50 = metric.CalculatePercentile(c.freeMbs, 50)

	return
}

func toMegabytes(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024)
}
