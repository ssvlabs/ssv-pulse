package infrastructure

import (
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

type CPUMonitor struct {
	user, system, total          uint64
	systemPercents, userPercents []float64
}

func NewCPU() *CPUMonitor {
	return &CPUMonitor{}
}

func (c *CPUMonitor) Measure() (systemPercent, userPercent float64, err error) {
	cpu, err := cpu.Get()
	if err != nil {
		return systemPercent, userPercent, err
	}
	systemPercent = float64(cpu.System-c.system) / float64(cpu.Total-c.total) * 100
	userPercent = float64(cpu.User-c.user) / float64(cpu.Total-c.total) * 100

	c.user = cpu.User
	c.system = cpu.System
	c.total = cpu.Total

	c.systemPercents = append(c.systemPercents, systemPercent)
	c.userPercents = append(c.userPercents, userPercent)

	return systemPercent, userPercent, nil
}

func (c *CPUMonitor) GetAggregatedValues() (userP50, systemP50 float64, total uint64) {
	userP50 = metric.CalculatePercentile(c.userPercents, 50)
	systemP50 = metric.CalculatePercentile(c.systemPercents, 50)

	return userP50, systemP50, c.total
}
