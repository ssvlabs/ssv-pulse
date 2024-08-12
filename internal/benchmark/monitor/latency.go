package monitor

import (
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/metric"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/network"
)

type (
	LatencyMonitor struct {
		min, max  time.Duration
		latencies []time.Duration
		network   network.Name
		listener  listenerSvc
	}
	Latency struct {
		Min, P10, P50, P90, Max time.Duration
	}
)

func NewLatency(listener listenerSvc, network network.Name) *LatencyMonitor {
	return &LatencyMonitor{
		listener: listener,
		network:  network,
	}
}

func (l *LatencyMonitor) Measure(slot phase0.Slot) Latency {
	receival, ok := l.listener.Receival(slot)
	if !ok {
		return Latency{}
	}

	latency := receival.Received.Sub(slotTime(network.GenesisTime[l.network], slot))

	if len(l.latencies) == 0 {
		l.min = latency
	} else if latency < l.min {
		l.min = latency
	}
	if latency > l.max {
		l.max = latency
	}

	l.latencies = append(l.latencies, latency)
	p10 := metric.CalculatePercentile(l.latencies, 10)
	p50 := metric.CalculatePercentile(l.latencies, 50)
	p90 := metric.CalculatePercentile(l.latencies, 90)

	return Latency{
		Min: l.min,
		P10: p10,
		P50: p50,
		P90: p90,
		Max: l.max,
	}
}

func slotTime(genesisTime time.Time, slot phase0.Slot) time.Time {
	return genesisTime.Add(time.Duration(slot) * 12 * time.Second)
}
