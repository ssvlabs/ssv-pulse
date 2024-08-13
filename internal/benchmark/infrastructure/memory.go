package infrastructure

import (
	"errors"

	"github.com/mackerelio/go-osstat/memory"
)

type MemoryMonitor struct {
}

func NewMemory() *MemoryMonitor {
	return &MemoryMonitor{}
}

func (MemoryMonitor) Measure() (totalMb, usedMb, cachedMb, freeMb float64, err error) {
	memory, err := memory.Get()
	if err != nil {
		return totalMb, usedMb, cachedMb, freeMb, errors.Join(err, errors.New("failed to measure memory metric"))
	}
	return toMegabytes(memory.Total), toMegabytes(memory.Used), toMegabytes(memory.Cached), toMegabytes(memory.Free), err
}

func toMegabytes(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024)
}
