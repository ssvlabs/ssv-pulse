package consensus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "pulse"
	subsystem = "consensus"

	serverAddrLabelName = "server_address"
)

var (
	labels = []string{serverAddrLabelName}

	peerCountMetric = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "peer_count",
			Help:      "number of peers",
			Namespace: namespace,
			Subsystem: subsystem,
		}, labels)

	latencyMetric = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:      "latency",
		Help:      "histogram of latencies for HTTP requests in seconds",
		Buckets:   prometheus.DefBuckets,
		Namespace: namespace,
		Subsystem: subsystem,
	}, labels)

	missedBlocksMetric = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "missed_blocks",
			Help:      "missed blocks",
			Namespace: namespace,
			Subsystem: subsystem,
		}, labels)

	receivedBlocksMetric = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "received_blocks",
			Help:      "received blocks",
			Namespace: namespace,
			Subsystem: subsystem,
		}, labels)

	missedAttestationsMetric = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "missed_attestations",
			Help:      "missed attestations",
			Namespace: namespace,
			Subsystem: subsystem,
		}, labels)

	freshAttestationsMetric = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "fresh_attestations",
			Help:      "fresh attestations",
			Namespace: namespace,
			Subsystem: subsystem,
		}, labels)

	correctnessMetric = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "correctness",
			Help:      "attestation correctness based on number of received blocks and attestations",
			Namespace: namespace,
			Subsystem: subsystem,
		}, labels)
)

func serverAddrLabel(serverAddr string) map[string]string {
	return map[string]string{
		serverAddrLabelName: serverAddr,
	}
}
