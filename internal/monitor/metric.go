package monitor

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/monitor/listener"
)

type (
	listenerSvc interface {
		Receival(slot phase0.Slot) (listener.SlotData, bool)
	}
	Type string
)

const (
	Latency Type = "Latency"
	Peers   Type = "Peers"
	Blocks  Type = "Blocks"
)
