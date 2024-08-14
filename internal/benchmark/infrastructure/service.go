package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	CPU    metric.Name = "CPU"
	Memory metric.Name = "Memory"
)

type Service struct {
	cpu      *CPUMetric
	memory   *MemoryMetric
	interval time.Duration
}

func New() *Service {
	return &Service{
		cpu:      NewCPUMetric(),
		memory:   NewMemoryMetric(),
		interval: time.Second * 5,
	}
}

func (s *Service) Start(ctx context.Context) (map[metric.Name]metric.Result, error) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			system, user, err := s.cpu.Get()
			if err != nil {
				logger.WriteError(metric.InfrastructureGroup, CPU, err)
			} else {
				logger.WriteMetric(metric.InfrastructureGroup, CPU, map[string]any{
					"system": system,
					"user":   user,
				})
			}

			total, used, cached, free, err := s.memory.Get()
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
			userP50, systemP50, total := s.cpu.Aggregate()
			totalP50, usedP50, cachedP50, freeP50 := s.memory.Aggregate()

			return map[metric.Name]metric.Result{
				CPU: {
					Value:    []byte(fmt.Sprintf("user_P50=%.2f%%, system_P50=%.2f%%, total=%v", userP50, systemP50, total)),
					Health:   "",
					Severity: "",
				},
				Memory: {
					Value:    []byte(fmt.Sprintf("total_P50=%.2fMB, used_P50=%.2fMB, cached_P50=%.2fMB, free_P50=%.2fMB", totalP50, usedP50, cachedP50, freeP50)),
					Health:   "",
					Severity: "",
				},
			}, ctx.Err()
		}
	}
}
