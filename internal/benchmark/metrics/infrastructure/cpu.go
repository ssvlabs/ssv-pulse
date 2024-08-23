package infrastructure

import (
	"context"
	"fmt"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	SystemCPUMeasurement = "System"
	UserCPUMeasurement   = "User"
)

type CPUMetric struct {
	metric.Base[float64]
	prevUser, prevSystem, total uint64
}

func NewCPUMetric(name string, healthCondition []metric.HealthCondition[float64]) *CPUMetric {
	return &CPUMetric{
		Base: metric.Base[float64]{
			Name:             name,
			HealthConditions: healthCondition,
		},
	}
}

func (c *CPUMetric) Measure(context.Context) {
	cpu, err := cpu.Get()
	if err != nil {
		logger.WriteError(metric.InfrastructureGroup, c.Name, err)
		return
	}
	systemPercent := float64(cpu.System-c.prevSystem) / float64(cpu.Total-c.total) * 100
	userPercent := float64(cpu.User-c.prevUser) / float64(cpu.Total-c.total) * 100

	c.prevUser = cpu.User
	c.prevSystem = cpu.System
	c.total = cpu.Total

	c.AddDataPoint(map[string]float64{
		SystemCPUMeasurement: systemPercent,
		UserCPUMeasurement:   userPercent,
	})

	logger.WriteMetric(metric.InfrastructureGroup, c.Name, map[string]any{
		SystemCPUMeasurement: systemPercent,
		UserCPUMeasurement:   userPercent,
	})
}

func (c *CPUMetric) AggregateResults() string {
	var values map[string][]float64 = make(map[string][]float64)

	for _, point := range c.DataPoints {
		values[SystemCPUMeasurement] = append(values[SystemCPUMeasurement], point.Values[SystemCPUMeasurement])
		values[UserCPUMeasurement] = append(values[UserCPUMeasurement], point.Values[UserCPUMeasurement])
	}

	return fmt.Sprintf("user_P50=%.2f%%, system_P50=%.2f%%, total=%v",
		metric.CalculatePercentile(values[UserCPUMeasurement], 50),
		metric.CalculatePercentile(values[SystemCPUMeasurement], 50), c.total)
}
