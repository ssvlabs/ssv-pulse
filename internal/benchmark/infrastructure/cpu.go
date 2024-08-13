package infrastructure

import (
	"github.com/mackerelio/go-osstat/cpu"
)

type CPUMonitor struct {
	user, system, total uint64
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

	return systemPercent, userPercent, nil
}
