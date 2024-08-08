package monitor

import (
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ssvlabsinfra/ssv-benchmark/configs"
)

type (
	LatencyMonitor struct {
		min, max, total time.Duration
		records         uint32
		network         configs.NetworkName
		listener        listenerSvc
	}
)

func NewLatency(listener listenerSvc, network configs.NetworkName) *LatencyMonitor {
	return &LatencyMonitor{
		listener: listener,
		network:  network,
	}
}

func (l *LatencyMonitor) Measure(slot phase0.Slot) (min, max, avg time.Duration) {
	receival, ok := l.listener.Receival(slot)
	if !ok {
		return l.min, l.max, l.total
	}

	latency := receival.Received.Sub(slotTime(configs.GenesisTime[l.network], slot))
	l.total += latency
	if l.records == 0 {
		l.min = latency
	} else if latency < l.min {
		l.min = latency
	}
	if latency > l.max {
		l.max = latency
	}
	l.records++
	return l.min, l.max, l.total / time.Duration(l.records)
}

func slotTime(genesisTime time.Time, slot phase0.Slot) time.Time {
	return genesisTime.Add(time.Duration(slot) * 12 * time.Second)
}
