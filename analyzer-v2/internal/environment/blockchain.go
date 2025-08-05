package environment

import (
	"fmt"
	"math"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type Blockchain struct {
	genesisTime  time.Time
	slotDuration time.Duration
}

func NewBlockchain(genesisTime time.Time, slotDuration time.Duration) *Blockchain {
	return &Blockchain{
		genesisTime:  genesisTime,
		slotDuration: slotDuration,
	}
}

// SlotStartTime returns the start time for the given slot.
func (b *Blockchain) SlotStartTime(slot phase0.Slot) (time.Time, error) {
	if slot > math.MaxInt64 {
		return time.Time{}, fmt.Errorf("slot number %d cannot exceed math.MaxInt64", slot)
	}
	maxAllowedSlotNumber := int64(math.MaxInt64 / b.slotDuration)
	if int64(slot) > maxAllowedSlotNumber {
		return time.Time{}, fmt.Errorf("slot number %d cannot exceed max allowed slot number of %d", slot, maxAllowedSlotNumber)
	}
	durationSinceGenesisStart := time.Duration(slot) * b.slotDuration
	start := b.genesisTime.Add(durationSinceGenesisStart)
	return start, nil
}

// EstimatedSlotAtTime estimates slot at the given time.
func (b *Blockchain) EstimatedSlotAtTime(time time.Time) (phase0.Slot, error) {
	if time.Before(b.genesisTime) {
		return phase0.Slot(0), fmt.Errorf("time %v is before genesis time %v", time, b.genesisTime)
	}
	timeAfterGenesis := time.Sub(b.genesisTime)
	return phase0.Slot(timeAfterGenesis / b.slotDuration), nil
}

var (
	BlockchainMainnet = NewBlockchain(
		time.Unix(1606824023, 0), // 2020-12-01 12:00:23 UTC
		12*time.Second,
	)
	BlockchainHoodi = NewBlockchain(
		time.Unix(1742213400, 0), // 2025-03-17 12:10:00 UTC
		12*time.Second,
	)
)

func BlockchainByName(name string) (*Blockchain, error) {
	if name == "mainnet" {
		return BlockchainMainnet, nil
	}
	if name == "hoodi" {
		return BlockchainHoodi, nil
	}
	return nil, fmt.Errorf("unknown blockchain: %s", name)
}
