package execution

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "pulse"
	subsystem = "execution"

	serverAddrLabelName = "server_address"
)

var (
	labels = []string{serverAddrLabelName}

	latencyMetric = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:      "latency",
		Help:      "histogram of latencies for HTTP requests in seconds",
		Buckets:   prometheus.DefBuckets,
		Namespace: namespace,
		Subsystem: subsystem,
	}, labels)

	peerCountMetric = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "peer_count",
			Help:      "number of peers",
			Namespace: namespace,
			Subsystem: subsystem,
		}, labels)
)

func serverAddrLabel(serverAddr string) map[string]string {
	return map[string]string{
		serverAddrLabelName: serverAddr,
	}
}
