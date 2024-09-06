package infrastructure

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace            = "pulse"
	subsystem            = "infrastructure"
	memoryUsageTypeLabel = "type"
	cpuUsageTypeLabel    = "type"
)

var (
	memoryUsageMetric = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "memory_usage",
			Help:      "memory usage",
			Namespace: namespace,
			Subsystem: subsystem,
		}, []string{memoryUsageTypeLabel})

	cpuUsageMetric = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "cpu_usage",
			Help:      "cpu usage",
			Namespace: namespace,
			Subsystem: subsystem,
		}, []string{cpuUsageTypeLabel})
)
