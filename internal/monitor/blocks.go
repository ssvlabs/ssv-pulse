package monitor

import "github.com/attestantio/go-eth2-client/spec/phase0"

type BlocksMonitor struct {
	listener       listenerSvc
	blocksReceived uint32
	blocksMissed   uint32
}

func NewBlocks(listener listenerSvc) *BlocksMonitor {
	return &BlocksMonitor{
		listener: listener,
	}
}

func (b *BlocksMonitor) Measure(slot phase0.Slot) (received, missed uint32) {
	_, ok := b.listener.Receival(slot)
	if !ok {
		b.blocksMissed++
	} else {
		b.blocksReceived++
	}
	return b.blocksReceived, b.blocksMissed
}
