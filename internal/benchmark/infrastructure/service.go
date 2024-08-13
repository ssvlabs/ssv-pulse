package infrastructure

import (
	"context"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	CPU    metric.Name = "CPU"
	Memory metric.Name = "Memory"
)

type Service struct {
	cpu      *CPUMonitor
	memory   *MemoryMonitor
	interval time.Duration
}

func New() *Service {
	return &Service{
		cpu:      NewCPU(),
		memory:   NewMemory(),
		interval: time.Second * 5,
	}
}

func (s *Service) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			system, user, err := s.cpu.Measure()
			if err != nil {
				logger.WriteError(metric.InfrastructureGroup, CPU, err)
			} else {
				logger.WriteMetric(metric.InfrastructureGroup, CPU, map[string]any{
					"system": system,
					"user":   user,
				})
			}

			total, used, cached, free, err := s.memory.Measure()
			if err != nil {
				logger.WriteError(metric.InfrastructureGroup, Memory, err)
			} else {
				logger.WriteMetric(metric.InfrastructureGroup, Memory, map[string]any{
					"total":  total,
					"used":   used,
					"cached": cached,
					"free":   free,
				})
			}

		case <-ctx.Done():
			return
		}
	}
}
