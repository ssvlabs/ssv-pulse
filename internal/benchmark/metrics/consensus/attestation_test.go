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
		finalizedAhead:        make(map[phase0.Slot]struct{}),
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

func TestGivenSlotAlreadyFinalizedWhenLateHeadEventArrivesThenStillRejected(t *testing.T) {
	a := newTestAttestationMetric()

	a.calculateMeasurements(phase0.Slot(100))
	a.calculateMeasurements(phase0.Slot(60)) // simulates an earlier slot's goroutine finishing late

	a.mu.Lock()
	isSlot60Finalized := a.isFinalized(phase0.Slot(60))
	isSlot100Finalized := a.isFinalized(phase0.Slot(100))
	a.mu.Unlock()
	assert.True(t, isSlot60Finalized)
	assert.True(t, isSlot100Finalized)

	// A head event for slot 60, arriving after slot 60 was itself already
	// finalized, must still be rejected.
	a.recordHeadEvent(phase0.Slot(60), phase0.Root{0x6})

	a.mu.Lock()
	_, ok := a.eventBlockRoots[phase0.Slot(60)]
	a.mu.Unlock()
	assert.False(t, ok)
}

// TestGivenLargeStartingSlotWhenFinalizingConsecutiveAndOutOfOrderSlotsThenWatermarkCompacts
// guards against finalizedWatermark's zero-value default silently becoming
// the effective baseline: a real benchmark's first finalized slot is
// genesisSlot+1, a mainnet-scale number in the tens of millions, not 1. If
// the watermark isn't initialized to that starting slot before the event
// listener starts (see Measure), every real slot lands in finalizedAhead
// forever, since the compaction loop can never find slot 1 there — the
// exact same unbounded growth this whole fix exists to prevent, just moved
// into a new field. Using small slot numbers relative to a zero-value
// watermark (as the other tests in this file do, for simplicity) would
// never have caught that, so this test explicitly starts from a
// realistic, large, nonzero baseline.
func TestGivenLargeStartingSlotWhenFinalizingConsecutiveAndOutOfOrderSlotsThenWatermarkCompacts(t *testing.T) {
	a := newTestAttestationMetric()

	const start = phase0.Slot(10_000_000) // mainnet-scale starting slot
	a.mu.Lock()
	a.finalizedWatermark = start
	a.mu.Unlock()

	// start+2 and start+3 finalize before start+1 (out of order).
	a.calculateMeasurements(start + 3)
	a.calculateMeasurements(start + 2)

	a.mu.Lock()
	watermark := a.finalizedWatermark
	aheadSize := len(a.finalizedAhead)
	a.mu.Unlock()
	assert.Equal(t, start, watermark, "watermark must not advance past a gap at start+1")
	assert.Equal(t, 2, aheadSize, "finalizedAhead must hold exactly the out-of-order slots seen so far, not grow unbounded")

	// The gap closes.
	a.calculateMeasurements(start + 1)

	a.mu.Lock()
	watermark = a.finalizedWatermark
	aheadSize = len(a.finalizedAhead)
	a.mu.Unlock()
	assert.Equal(t, start+3, watermark, "watermark must compact through the whole contiguous run once the gap closes")
	assert.Equal(t, 0, aheadSize, "finalizedAhead must be empty again once compaction catches up")
}

func TestGivenHigherSlotFinalizesFirstWhenLowerSlotDataArrivesThenAcceptedNotMissed(t *testing.T) {
	a := newTestAttestationMetric()

	// Give slot 100 its own event/attestation so its own finalization is a
	// legitimate "received" rather than a "missed" — keeping the assertions
	// below focused purely on whether slot 99 gets miscounted because of it.
	slot100Root := phase0.Root{0x64}
	a.recordHeadEvent(phase0.Slot(100), slot100Root)
	a.recordAttestationRoot(phase0.Slot(100), slot100Root)

	// Slot 100's own goroutine finishes first (e.g. its fetchAttestationData
	// call happened to be faster), finalizing slot 100 before slot 99 has
	// been finalized at all.
	a.calculateMeasurements(phase0.Slot(100))

	a.mu.Lock()
	isSlot99FinalizedTooEarly := a.isFinalized(phase0.Slot(99))
	a.mu.Unlock()
	assert.False(t, isSlot99FinalizedTooEarly, "slot 99 must not be considered finalized just because a higher slot finalized first")

	// Slot 99's on-time head event and attestation root now arrive. With a
	// single scalar "highest slot seen" watermark these would have been
	// incorrectly rejected, since 99 <= 100; they must be accepted here.
	root := phase0.Root{0x63}
	a.recordHeadEvent(phase0.Slot(99), root)
	a.recordAttestationRoot(phase0.Slot(99), root)

	a.calculateMeasurements(phase0.Slot(99))

	assert.Equal(t, uint64(2), a.receivedBlocksCount.Load(), "both slot 99 and slot 100 must be counted as received")
	assert.Equal(t, uint64(0), a.missedBlocksCount.Load(), "slot 99 must not be counted as missed just because slot 100 finalized first")
	assert.Equal(t, uint64(2), a.freshAttestationsCount.Load())
	assert.Equal(t, uint64(0), a.missedAttestationsCount.Load())
}

// TestGivenConcurrentHeadEventsAndFinalizationWhenRacingThenNoEntriesLeakOrMiscount
// exercises the actual races the previous implementations were vulnerable
// to: recordHeadEvent and calculateMeasurements firing for many different
// slots concurrently, with no guaranteed ordering either within a slot or
// across slots. Run with -race. Beyond confirming nothing leaks, this also
// checks that every slot is accounted for exactly once — a slot's data
// racing with a *different*, out-of-order slot's finalization must never
// be silently dropped (see TestGivenHigherSlotFinalizesFirstWhenLowerSlotDataArrivesThenAcceptedNotMissed
// for the deterministic version of that specific scenario).
func TestGivenConcurrentHeadEventsAndFinalizationWhenRacingThenNoEntriesLeakOrMiscount(t *testing.T) {
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

	// calculateMeasurements runs exactly once per slot, and each run counts
	// the slot as exactly one of received or missed — regardless of race
	// outcome, the two counters must sum to the total number of slots, with
	// no slot double-counted or dropped.
	assert.Equal(t, uint64(slots), a.receivedBlocksCount.Load()+a.missedBlocksCount.Load())
}
