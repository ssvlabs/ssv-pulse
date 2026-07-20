package consensus

import (
	"sync"
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
		client:                fakeClientService{address: "fake-addr"},
		eventBlockRoots:       make(map[phase0.Slot]SlotData),
		attestationBlockRoots: make(map[phase0.Slot]phase0.Root),
	}
}

func TestGivenMissedBlockWhenCalculateMeasurementsThenAttestationRootIsNotLeaked(t *testing.T) {
	a := newTestAttestationMetric()

	const slot = phase0.Slot(10)

	// Simulates fetchAttestationData having already succeeded for this slot
	// before the corresponding head event turned out to be missing.
	a.attestationBlockRoots[slot] = phase0.Root{0x1}

	a.calculateMeasurements(slot)

	a.mu.Lock()
	_, attestationRootRemains := a.attestationBlockRoots[slot]
	_, eventRootRemains := a.eventBlockRoots[slot]
	a.mu.Unlock()

	assert.False(t, attestationRootRemains, "attestationBlockRoots entry must not outlive a missed block")
	assert.False(t, eventRootRemains)

	assert.Equal(t, uint64(1), a.missedBlocksCount.Load())
	assert.Equal(t, uint64(0), a.receivedBlocksCount.Load())
}

func TestGivenReceivedBlockAndFreshAttestationWhenCalculateMeasurementsThenBothMapsAreConsumed(t *testing.T) {
	a := newTestAttestationMetric()

	const slot = phase0.Slot(20)
	root := phase0.Root{0x2}

	a.eventBlockRoots[slot] = SlotData{RootBlock: root}
	a.attestationBlockRoots[slot] = root

	a.calculateMeasurements(slot)

	a.mu.Lock()
	_, eventRootRemains := a.eventBlockRoots[slot]
	_, attestationRootRemains := a.attestationBlockRoots[slot]
	a.mu.Unlock()

	assert.False(t, eventRootRemains)
	assert.False(t, attestationRootRemains)

	assert.Equal(t, uint64(1), a.receivedBlocksCount.Load())
	assert.Equal(t, uint64(1), a.freshAttestationsCount.Load())
	assert.Equal(t, uint64(0), a.missedAttestationsCount.Load())
}

func TestGivenReceivedBlockButMissingAttestationWhenCalculateMeasurementsThenMapsAreConsumed(t *testing.T) {
	a := newTestAttestationMetric()

	const slot = phase0.Slot(30)

	a.eventBlockRoots[slot] = SlotData{RootBlock: phase0.Root{0x3}}
	// No corresponding attestationBlockRoots entry: attestation was missed.

	a.calculateMeasurements(slot)

	a.mu.Lock()
	_, eventRootRemains := a.eventBlockRoots[slot]
	a.mu.Unlock()

	assert.False(t, eventRootRemains)

	assert.Equal(t, uint64(1), a.receivedBlocksCount.Load())
	assert.Equal(t, uint64(1), a.missedAttestationsCount.Load())
	assert.Equal(t, uint64(0), a.freshAttestationsCount.Load())
}

func TestGivenOnTimeHeadEventWhenRecordHeadEventThenStored(t *testing.T) {
	a := newTestAttestationMetric()

	const slot = phase0.Slot(40)
	a.recordHeadEvent(slot, phase0.Root{0x4})

	a.mu.Lock()
	stored, ok := a.eventBlockRoots[slot]
	a.mu.Unlock()

	assert.True(t, ok)
	assert.Equal(t, phase0.Root{0x4}, stored.RootBlock)
}

func TestGivenLateHeadEventForAlreadyProcessedSlotWhenRecordHeadEventThenNotStored(t *testing.T) {
	a := newTestAttestationMetric()

	const slot = phase0.Slot(50)
	a.calculateMeasurements(slot) // no event yet; advances the watermark to 50, records a missed block

	a.recordHeadEvent(slot, phase0.Root{0x5}) // arrives too late

	a.mu.Lock()
	_, ok := a.eventBlockRoots[slot]
	a.mu.Unlock()

	assert.False(t, ok, "a late head event for an already-processed slot must not be stored")
}

func TestGivenOutOfOrderCalculateMeasurementsWhenFinalizeSlotThenWatermarkNeverMovesBackwards(t *testing.T) {
	a := newTestAttestationMetric()

	a.calculateMeasurements(phase0.Slot(100))
	a.calculateMeasurements(phase0.Slot(60)) // simulates an earlier slot's goroutine finishing late

	a.mu.Lock()
	watermark := a.lastProcessedSlot
	a.mu.Unlock()
	assert.Equal(t, phase0.Slot(100), watermark)

	// A head event for slot 60, arriving after slot 100 was already
	// finalized, must still be rejected even though it was processed
	// "out of order" relative to slot 60's own goroutine.
	a.recordHeadEvent(phase0.Slot(60), phase0.Root{0x6})

	a.mu.Lock()
	_, ok := a.eventBlockRoots[phase0.Slot(60)]
	a.mu.Unlock()
	assert.False(t, ok)
}

// TestGivenConcurrentHeadEventsAndFinalizationWhenRacingThenNoEntriesLeak
// exercises the actual race the previous sync.Map + atomic-watermark
// implementation was vulnerable to: recordHeadEvent and calculateMeasurements
// firing for the same slot at the same time, with no guaranteed ordering.
// Run with -race; the assertion afterwards additionally confirms that
// regardless of which one "won" for a given slot, no map entries are left
// behind.
func TestGivenConcurrentHeadEventsAndFinalizationWhenRacingThenNoEntriesLeak(t *testing.T) {
	a := newTestAttestationMetric()

	const slots = 500

	var wg sync.WaitGroup
	for i := phase0.Slot(1); i <= slots; i++ {
		wg.Add(2)
		go func(slot phase0.Slot) {
			defer wg.Done()
			a.recordHeadEvent(slot, phase0.Root{byte(slot)})
		}(i)
		go func(slot phase0.Slot) {
			defer wg.Done()
			a.calculateMeasurements(slot)
		}(i)
	}
	wg.Wait()

	a.mu.Lock()
	remaining := len(a.eventBlockRoots) + len(a.attestationBlockRoots)
	a.mu.Unlock()

	assert.Equal(t, 0, remaining, "no entries should remain once every slot has been both recorded and finalized")
}
