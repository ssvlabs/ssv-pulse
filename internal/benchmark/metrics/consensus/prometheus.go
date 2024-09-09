package consensus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "pulse"
	subsystem = "consensus"
)

var (
	peerCountMetric = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name:      "peer_count",
			Help:      "number of peers",
			Namespace: namespace,
			Subsystem: subsystem,
		})

	latencyMetric = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:      "latency",
		Help:      "histogram of latencies for HTTP requests in seconds",
		Buckets:   prometheus.DefBuckets,
		Namespace: namespace,
		Subsystem: subsystem,
	})

	missedBlocksMetric = promauto.NewCounter(
		prometheus.CounterOpts{
			Name:      "missed_blocks",
			Help:      "missed blocks",
			Namespace: namespace,
			Subsystem: subsystem,
		})

	receivedBlocksMetric = promauto.NewCounter(
		prometheus.CounterOpts{
			Name:      "received_blocks",
			Help:      "received blocks",
			Namespace: namespace,
			Subsystem: subsystem,
		})

	missedAttestationsMetric = promauto.NewCounter(
		prometheus.CounterOpts{
			Name:      "missed_attestations",
			Help:      "missed attestations",
			Namespace: namespace,
			Subsystem: subsystem,
		})

	freshAttestationsMetric = promauto.NewCounter(
		prometheus.CounterOpts{
			Name:      "fresh_attestations",
			Help:      "fresh attestations",
			Namespace: namespace,
			Subsystem: subsystem,
		})

	correctnessMetric = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name:      "correctness",
			Help:      "attestation correctness based on number of received blocks and attestations",
			Namespace: namespace,
			Subsystem: subsystem,
		})
)
