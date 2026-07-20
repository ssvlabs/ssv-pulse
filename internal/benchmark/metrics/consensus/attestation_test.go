package consensus

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
)

type fakeClientService struct {
	address string
}

func (f fakeClientService) Name() string    { return "fake" }
func (f fakeClientService) Address() string { return f.address }
func (f fakeClientService) IsActive() bool  { return true }
func (f fakeClientService) IsSynced() bool  { return true }

func newTestAttestationMetric() *AttestationMetric {
	return &AttestationMetric{
		client: fakeClientService{address: "fake-addr"},
	}
}

func TestGivenMissedBlockWhenCalculateMeasurementsThenAttestationRootIsNotLeaked(t *testing.T) {
	a := newTestAttestationMetric()

	const slot = phase0.Slot(10)

	// Simulates fetchAttestationData having already succeeded for this slot
	// before the corresponding head event turned out to be missing.
	a.attestationBlockRoots.Store(slot, phase0.Root{0x1})

	a.calculateMeasurements(slot)

	_, attestationRootRemains := a.attestationBlockRoots.Load(slot)
	assert.False(t, attestationRootRemains, "attestationBlockRoots entry must not outlive a missed block")

	_, eventRootRemains := a.eventBlockRoots.Load(slot)
	assert.False(t, eventRootRemains)

	assert.Equal(t, uint64(1), a.missedBlocksCount.Load())
	assert.Equal(t, uint64(0), a.receivedBlocksCount.Load())
}

func TestGivenReceivedBlockAndFreshAttestationWhenCalculateMeasurementsThenBothMapsAreConsumed(t *testing.T) {
	a := newTestAttestationMetric()

	const slot = phase0.Slot(20)
	root := phase0.Root{0x2}

	a.eventBlockRoots.Store(slot, SlotData{RootBlock: root})
	a.attestationBlockRoots.Store(slot, root)

	a.calculateMeasurements(slot)

	_, eventRootRemains := a.eventBlockRoots.Load(slot)
	assert.False(t, eventRootRemains)
	_, attestationRootRemains := a.attestationBlockRoots.Load(slot)
	assert.False(t, attestationRootRemains)

	assert.Equal(t, uint64(1), a.receivedBlocksCount.Load())
	assert.Equal(t, uint64(1), a.freshAttestationsCount.Load())
	assert.Equal(t, uint64(0), a.missedAttestationsCount.Load())
}

func TestGivenReceivedBlockButMissingAttestationWhenCalculateMeasurementsThenMapsAreConsumed(t *testing.T) {
	a := newTestAttestationMetric()

	const slot = phase0.Slot(30)

	a.eventBlockRoots.Store(slot, SlotData{RootBlock: phase0.Root{0x3}})
	// No corresponding attestationBlockRoots entry: attestation was missed.

	a.calculateMeasurements(slot)

	_, eventRootRemains := a.eventBlockRoots.Load(slot)
	assert.False(t, eventRootRemains)

	assert.Equal(t, uint64(1), a.receivedBlocksCount.Load())
	assert.Equal(t, uint64(1), a.missedAttestationsCount.Load())
	assert.Equal(t, uint64(0), a.freshAttestationsCount.Load())
}

func TestGivenOnTimeHeadEventWhenRecordHeadEventThenStored(t *testing.T) {
	a := newTestAttestationMetric()

	const slot = phase0.Slot(40)
	a.recordHeadEvent(slot, phase0.Root{0x4})

	stored, ok := a.eventBlockRoots.Load(slot)
	assert.True(t, ok)
	assert.Equal(t, phase0.Root{0x4}, stored.(SlotData).RootBlock)
}

func TestGivenLateHeadEventForAlreadyProcessedSlotWhenRecordHeadEventThenNotStored(t *testing.T) {
	a := newTestAttestationMetric()

	const slot = phase0.Slot(50)
	a.calculateMeasurements(slot) // no event yet; advances lastProcessedSlot to 50, records a missed block

	a.recordHeadEvent(slot, phase0.Root{0x5}) // arrives too late

	_, ok := a.eventBlockRoots.Load(slot)
	assert.False(t, ok, "a late head event for an already-processed slot must not be stored")
}

func TestGivenOutOfOrderCalculateMeasurementsWhenAdvanceLastProcessedSlotThenWatermarkNeverMovesBackwards(t *testing.T) {
	a := newTestAttestationMetric()

	a.calculateMeasurements(phase0.Slot(100))
	a.calculateMeasurements(phase0.Slot(60)) // simulates an earlier slot's goroutine finishing late

	assert.Equal(t, uint64(100), a.lastProcessedSlot.Load())

	// A head event for slot 60, arriving after slot 100 was already
	// finalized, must still be rejected even though it was processed
	// "out of order" relative to slot 60's own goroutine.
	a.recordHeadEvent(phase0.Slot(60), phase0.Root{0x6})
	_, ok := a.eventBlockRoots.Load(phase0.Slot(60))
	assert.False(t, ok)
}
