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
	peerCountMetric = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name:      "peer_count",
			Help:      "number of peers",
			Namespace: namespace,
			Subsystem: subsystem,
		})
)
