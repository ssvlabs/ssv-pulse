package infrastructure

import (
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

func (m *CPUMetric) Measure() {
	cpu, err := cpu.Get()
	if err != nil {
		logger.WriteError(metric.InfrastructureGroup, m.Name, err)
		return
	}
	systemPercent := float64(cpu.System-m.prevSystem) / float64(cpu.Total-m.total) * 100
	userPercent := float64(cpu.User-m.prevUser) / float64(cpu.Total-m.total) * 100

	m.prevUser = cpu.User
	m.prevSystem = cpu.System
	m.total = cpu.Total

	m.AddDataPoint(map[string]float64{
		SystemCPUMeasurement: systemPercent,
		UserCPUMeasurement:   userPercent,
	})

	logger.WriteMetric(metric.InfrastructureGroup, m.Name, map[string]any{
		"system": systemPercent,
		"user":   userPercent,
	})
}

func (p *CPUMetric) AggregateResults() string {
	var values map[string][]float64 = make(map[string][]float64)

	for _, point := range p.DataPoints {
		values[SystemCPUMeasurement] = append(values[SystemCPUMeasurement], point.Values[SystemCPUMeasurement])
		values[UserCPUMeasurement] = append(values[UserCPUMeasurement], point.Values[UserCPUMeasurement])
	}

	return fmt.Sprintf("user_P50=%.2f%%, system_P50=%.2f%%, total=%v",
		metric.CalculatePercentile(values[UserCPUMeasurement], 50),
		metric.CalculatePercentile(values[SystemCPUMeasurement], 50), p.total)
}
