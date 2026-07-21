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

func TestGivenLateHeadEventForAlreadyFinalizedSlotThenNotStored(t *testing.T) {
	a := newTestAttestationMetric()

	const slot = phase0.Slot(50)
	a.calculateMeasurements(slot) // no event yet; finalizes the slot, records a missed block

	a.recordHeadEvent(slot, phase0.Root{0x5}) // arrives too late

	a.mu.Lock()
	_, ok := a.eventBlockRoots[slot]
	a.mu.Unlock()

	assert.False(t, ok, "a late head event for an already-finalized slot must not be stored")
}

// TestGivenSequentialFinalizationThenWatermarkAdvancesAndMapsStayEmpty covers
// the memory-bound property directly: finalizing a long contiguous run of
// slots (in the strictly increasing order the scheduler uses) leaves nothing
// behind. A large starting slot mirrors production, where the first finalized
// slot is genesisSlot+1 (tens of millions), not 1 — the scalar watermark
// handles that with no per-slot state to accumulate.
func TestGivenSequentialFinalizationThenWatermarkAdvancesAndMapsStayEmpty(t *testing.T) {
	a := newTestAttestationMetric()

	const (
		start = phase0.Slot(10_000_000) // mainnet-scale starting slot
		count = 1_000
	)

	for slot := start; slot < start+count; slot++ {
		root := phase0.Root{byte(slot)}
		a.recordHeadEvent(slot, root)
		a.recordAttestationRoot(slot, root)
		a.calculateMeasurements(slot)
	}

	a.mu.Lock()
	watermark := a.finalizedSlot
	remaining := len(a.eventBlockRoots) + len(a.attestationBlockRoots)
	a.mu.Unlock()

	assert.Equal(t, start+count-1, watermark, "watermark must track the highest finalized slot")
	assert.Equal(t, 0, remaining, "no per-slot state may accumulate across a long run")
	assert.Equal(t, uint64(count), a.receivedBlocksCount.Load())
	assert.Equal(t, uint64(0), a.missedBlocksCount.Load())
}

// TestGivenConcurrentRecordsAndSequentialFinalizationThenNoLeakOrMiscount
// reproduces the production concurrency shape: a single goroutine finalizes
// slots in increasing order (the scheduler), while head events and
// attestation roots for those slots are recorded concurrently from other
// goroutines (the event listener and async fetches). Run with -race. Whether
// a given slot's record wins or loses its race with that slot's finalization
// is nondeterministic, so we only assert the invariants that must always
// hold: nothing leaks, and every slot is finalized exactly once (received +
// missed == total).
func TestGivenConcurrentRecordsAndSequentialFinalizationThenNoLeakOrMiscount(t *testing.T) {
	a := newTestAttestationMetric()

	const slots = 500

	var wg sync.WaitGroup

	// Concurrent recorders, one per slot.
	for i := phase0.Slot(1); i <= slots; i++ {
		wg.Add(1)
		go func(slot phase0.Slot) {
			defer wg.Done()
			root := phase0.Root{byte(slot)}
			a.recordHeadEvent(slot, root)
			a.recordAttestationRoot(slot, root)
		}(i)
	}

	// Single sequential finalizer, mirroring the scheduler goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for slot := phase0.Slot(1); slot <= slots; slot++ {
			a.calculateMeasurements(slot)
		}
	}()

	wg.Wait()

	a.mu.Lock()
	remaining := len(a.eventBlockRoots) + len(a.attestationBlockRoots)
	a.mu.Unlock()
	assert.Equal(t, 0, remaining, "no entries should remain once every slot has been recorded and finalized")

	assert.Equal(t, uint64(slots), a.receivedBlocksCount.Load()+a.missedBlocksCount.Load(),
		"every slot must be finalized exactly once, counted as received or missed but never both or neither")
}
