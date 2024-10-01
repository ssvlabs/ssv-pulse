package execution

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "pulse"
	subsystem = "execution"
)

var (
	latencyMetric = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:      "latency",
		Help:      "histogram of latencies for HTTP requests in seconds",
		Buckets:   prometheus.DefBuckets,
		Namespace: namespace,
		Subsystem: subsystem,
	})

	peerCountMetric = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name:      "peer_count",
			Help:      "number of peers",
			Namespace: namespace,
			Subsystem: subsystem,
		})
)
