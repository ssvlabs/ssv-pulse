package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/ssvlabs/ssv-pulse/internal/platform/logger"
	"github.com/ssvlabs/ssv-pulse/internal/platform/metric"
)

const (
	SystemCPUMeasurement = "System"
	UserCPUMeasurement   = "User"
)

type CPUMetric struct {
	metric.Base[float64]
	// prevUser/prevSystem are only ever touched by the measure goroutine.
	// total is also read by AggregateResults, which runs at shutdown without
	// waiting for that goroutine to stop, so it must be synchronized.
	prevUser, prevSystem uint64
	total                atomic.Uint64
	interval             time.Duration
	systemHistogram      *metric.Histogram[float64]
	userHistogram        *metric.Histogram[float64]
}

func NewCPUMetric(name string, interval time.Duration, healthCondition []metric.HealthCondition[float64]) *CPUMetric {
	return &CPUMetric{
		Base: metric.Base[float64]{
			Name:             name,
			HealthConditions: healthCondition,
		},
		interval:        interval,
		systemHistogram: metric.NewHistogram[float64](),
		userHistogram:   metric.NewHistogram[float64](),
	}
}

// roundPercent buckets a percentage to 2-decimal precision (matching this
// metric's display precision) so the histogram's cardinality stays bounded
// by the value domain (0-100) instead of growing with every observation.
func roundPercent(v float64) float64 {
	return math.Round(v*100) / 100
}

func (c *CPUMetric) Measure(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.With("metric_name", c.Name).Debug("metric was stopped")
			return
		case <-ticker.C:
			c.measure()
		}
	}
}

func (c *CPUMetric) measure() {
	cpu, err := cpu.Get()
	if err != nil {
		logger.WriteError(metric.InfrastructureGroup, c.Name, err)
		return
	}
	totalDelta := cpu.Total - c.total.Load()
	systemPercent := float64(cpu.System-c.prevSystem) / float64(totalDelta) * 100
	userPercent := float64(cpu.User-c.prevUser) / float64(totalDelta) * 100

	c.prevUser = cpu.User
	c.prevSystem = cpu.System
	c.total.Store(cpu.Total)

	c.writeMetric(systemPercent, userPercent)
}

func (c *CPUMetric) writeMetric(systemPercent, userPercent float64) {
	c.systemHistogram.Observe(roundPercent(systemPercent))
	c.userHistogram.Observe(roundPercent(userPercent))

	c.AddDataPoint(map[string]float64{
		SystemCPUMeasurement: systemPercent,
		UserCPUMeasurement:   userPercent,
	})

	cpuUsageMetric.With(prometheus.Labels{cpuUsageTypeLabel: "system"}).Set(float64(systemPercent))
	cpuUsageMetric.With(prometheus.Labels{cpuUsageTypeLabel: "user"}).Set(float64(userPercent))

	logger.WriteMetric(metric.InfrastructureGroup, c.Name, map[string]any{
		SystemCPUMeasurement: systemPercent,
		UserCPUMeasurement:   userPercent,
	})
}

func (c *CPUMetric) AggregateResults() string {
	return fmt.Sprintf("user_P50=%.2f%%, system_P50=%.2f%%, total=%v",
		c.userHistogram.Percentiles(50)[50],
		c.systemHistogram.Percentiles(50)[50], c.total.Load())
}
