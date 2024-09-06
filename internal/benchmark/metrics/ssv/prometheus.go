package ssv

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace                = "pulse"
	subsystem                = "ssv"
	connectionDirectionLabel = "direction"
)

var (
	peerCountMetric = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name:      "peer_count",
			Help:      "number of peers",
			Namespace: namespace,
			Subsystem: subsystem,
		})

	connectionsMetric = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "connections_count",
			Help:      "number of connection",
			Namespace: namespace,
			Subsystem: subsystem,
		}, []string{connectionDirectionLabel})
)
