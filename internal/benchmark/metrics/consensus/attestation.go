package consensus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/rs/zerolog"

	"github.com/ssvlabs/ssv-pulse/internal/platform/logger"
	"github.com/ssvlabs/ssv-pulse/internal/platform/metric"
)

const (
	blockMintingTime             = time.Second * 12
	unreadyBlockDelay            = time.Millisecond * 200
	MissedBlockMeasurement       = "MissedBlock"
	ReceivedBlockMeasurement     = "ReceivedBlock"
	MissedAttestationMeasurement = "MissedAttestation"
	FreshAttestationMeasurement  = "FreshAttestation"
	CorrectnessMeasurement       = "Correctness"
)

var (
	UnreadyBlockMeasurement = fmt.Sprintf("UnreadyBlockMeasurement%dms", unreadyBlockDelay/time.Millisecond)
)

type (
	SlotData struct {
		Received  time.Time
		RootBlock phase0.Root
	}

	AttestationMetric struct {
		metric.Base[float64]
		client      client.Service
		genesisTime time.Time

		// mu guards eventBlockRoots, attestationBlockRoots, and the
		// finalization state below together so a slot's finalization can
		// never interleave with a concurrent write for that same slot. An
		// earlier version synchronized the maps (sync.Map) and a single
		// scalar watermark (atomic.Uint64) independently, which left a
		// check-then-act window: a writer could read the watermark before
		// finalization advanced it, then write after finalization had
		// already decided the slot was missed, leaking permanently.
		//
		// finalizedWatermark/finalizedAhead track exactly which slots have
		// been finalized rather than just "the highest slot seen so far".
		// Per-slot goroutines can finalize out of order (fetchAttestationData
		// has variable network latency): a single scalar watermark would
		// then reject valid, still-pending data for a *lower*, not-yet-
		// finalized slot just because a *higher* slot happened to finish
		// first — turning a genuinely on-time event into a false "missed
		// block". finalizedWatermark is the highest slot with no gaps below
		// it; finalizedAhead holds slots finalized ahead of that contiguous
		// frontier, and is compacted into the watermark as soon as the gap
		// closes, so it only grows with actual observed out-of-order skew,
		// never with total runtime.
		mu                    sync.Mutex
		eventBlockRoots       map[phase0.Slot]SlotData
		attestationBlockRoots map[phase0.Slot]phase0.Root
		finalizedWatermark    phase0.Slot
		finalizedAhead        map[phase0.Slot]struct{}

		// Running counters backing AggregateResults, updated at the same
		// points AddDataPoint is called below. Kept separately from
		// Base[T] because Base no longer retains full history. Plain
		// atomics rather than mu: each is an independent always-+1
		// counter with no invariant linking it to the maps above.
		missedBlocksCount       atomic.Uint64
		receivedBlocksCount     atomic.Uint64
		missedAttestationsCount atomic.Uint64
		freshAttestationsCount  atomic.Uint64
		unreadyBlocksCount      atomic.Uint64
	}
)

func NewAttestationMetric(addr, name string, genesisTime time.Time, healthCondition []metric.HealthCondition[float64]) *AttestationMetric {
	client, err := http.New(
		context.TODO(),
		http.WithLogLevel(zerolog.DebugLevel),
		http.WithAddress(addr),
	)
	if err != nil {
		slog.
			With("addr", addr).
			Error("failed to instantiate Consensus Client")
		panic(err.Error())
	}

	return &AttestationMetric{
		Base: metric.Base[float64]{
			HealthConditions: healthCondition,
			Name:             name,
		},
		client:                client,
		eventBlockRoots:       make(map[phase0.Slot]SlotData),
		attestationBlockRoots: make(map[phase0.Slot]phase0.Root),
		finalizedAhead:        make(map[phase0.Slot]struct{}),
		genesisTime:           genesisTime,
	}
}

func (a *AttestationMetric) Measure(ctx context.Context) {
	genesisSlot := currentSlot(a.genesisTime)

	// finalizedWatermark defaults to zero, but the first slot this process
	// will ever finalize is genesisSlot+1 (a real network's genesis time is
	// years in the past, so this is a large number, not 1). Without this,
	// every real slot would be treated as "ahead" of a watermark stuck at
	// zero — the compaction loop in markFinalized would never find slot 1
	// in finalizedAhead to unblock it, and finalizedAhead would grow by one
	// entry per slot for the life of the process. Set before launchListener
	// starts, so no head event can be evaluated against the wrong baseline.
	a.mu.Lock()
	a.finalizedWatermark = genesisSlot
	a.mu.Unlock()

	go a.launchListener(ctx)

	go func() {
		slot := genesisSlot
		for {
			slot++
			nextSlotWithDelay := time.After(time.Until(slotTime(a.genesisTime, slot).Add(time.Second * 4)))
			select {
			case <-nextSlotWithDelay:
				go func(slot phase0.Slot) {
					a.fetchAttestationData(ctx, slot)
					const calculationSlotLag = 2
					if slot > genesisSlot+calculationSlotLag {
						a.calculateMeasurements(slot - calculationSlotLag)
					}
				}(slot)
			case <-ctx.Done():
				slog.With("metric_name", a.Name).Debug("metric was stopped")
				return
			}
		}
	}()
}

func (a *AttestationMetric) fetchAttestationData(ctx context.Context, slot phase0.Slot) {
	blockRoot, err := a.fetchAttestationBlockRoot(ctx, slot)

	if err != nil {
		logger.WriteError(metric.ConsensusGroup, a.Name, err)
		return
	}

	a.recordAttestationRoot(slot, blockRoot)
}

func (a *AttestationMetric) launchListener(ctx context.Context) {
	if err := a.client.(client.EventsProvider).Events(ctx, &api.EventsOpts{
		Topics: []string{"head"},
		Handler: func(event *v1.Event) {
			data := event.Data.(*v1.HeadEvent)

			a.recordHeadEvent(data.Slot, data.Block)

			go a.checkUnreadyBlock(ctx, data.Slot, data.Block)
		},
	}); err != nil {
		logger.WriteError(metric.ConsensusGroup, a.Name, err)
	}
}

// recordHeadEvent stores the block root observed for a slot's head event,
// unless that exact slot has already been finalized — a late event for an
// already-processed slot would never be consumed and would otherwise leak a
// permanent entry into eventBlockRoots. Sharing mu with finalizeSlot makes
// this check-and-store atomic with finalization: there is no window in
// which finalizeSlot can decide the slot was missed and then have this
// method write into it afterwards. Checking isFinalized(slot) rather than a
// single scalar watermark means a higher slot finalizing first can never
// cause this slot's legitimate, still-pending data to be rejected.
func (a *AttestationMetric) recordHeadEvent(slot phase0.Slot, block phase0.Root) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isFinalized(slot) {
		return
	}

	a.eventBlockRoots[slot] = SlotData{
		Received:  time.Now(),
		RootBlock: block,
	}
}

// recordAttestationRoot stores the attestation root fetched for a slot,
// guarded the same way and for the same reason as recordHeadEvent.
func (a *AttestationMetric) recordAttestationRoot(slot phase0.Slot, blockRoot phase0.Root) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isFinalized(slot) {
		return
	}

	a.attestationBlockRoots[slot] = blockRoot
}

// finalizeSlot marks slot as finalized and consumes both maps' entries for
// it in one critical section, so a concurrent recordHeadEvent or
// recordAttestationRoot can never write a new entry for this slot after
// finalization has already decided it.
func (a *AttestationMetric) finalizeSlot(slot phase0.Slot) (event SlotData, hasEvent bool, attestationRoot phase0.Root, hasAttestation bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	event, hasEvent = a.eventBlockRoots[slot]
	delete(a.eventBlockRoots, slot)

	attestationRoot, hasAttestation = a.attestationBlockRoots[slot]
	delete(a.attestationBlockRoots, slot)

	a.markFinalized(slot)

	return event, hasEvent, attestationRoot, hasAttestation
}

// markFinalized records slot as finalized, then compacts finalizedWatermark
// forward through any slots in finalizedAhead that are now contiguous with
// it. Per-slot goroutines can finalize out of order (fetchAttestationData
// has variable network latency), so a slot arriving before the watermark
// has caught up to it goes into finalizedAhead instead of advancing the
// watermark directly — keeping that set bounded by the actual out-of-order
// skew observed, not by total runtime, since entries are removed as soon as
// the gap below them closes.
func (a *AttestationMetric) markFinalized(slot phase0.Slot) {
	if slot <= a.finalizedWatermark {
		return
	}

	if slot == a.finalizedWatermark+1 {
		a.finalizedWatermark = slot
	} else {
		a.finalizedAhead[slot] = struct{}{}
	}

	for {
		next := a.finalizedWatermark + 1
		if _, ok := a.finalizedAhead[next]; !ok {
			break
		}
		delete(a.finalizedAhead, next)
		a.finalizedWatermark = next
	}
}

// isFinalized reports whether slot has already been finalized. Callers
// must hold mu.
func (a *AttestationMetric) isFinalized(slot phase0.Slot) bool {
	if slot <= a.finalizedWatermark {
		return true
	}
	_, ok := a.finalizedAhead[slot]
	return ok
}

func (a *AttestationMetric) checkUnreadyBlock(ctx context.Context, slot phase0.Slot, block phase0.Root) {
	time.Sleep(unreadyBlockDelay)
	blockRoot, err := a.fetchAttestationBlockRoot(ctx, slot)
	if err != nil {
		logger.WriteError(metric.ConsensusGroup, a.Name, err)
		return
	}

	if blockRoot != block {
		a.unreadyBlocksCount.Add(1)
		a.AddDataPoint(map[string]float64{
			UnreadyBlockMeasurement: 1,
		})
		logger.WriteMetric(metric.ConsensusGroup, a.Name, map[string]any{
			UnreadyBlockMeasurement: 1,
		})
	}
}

func (a *AttestationMetric) fetchAttestationBlockRoot(ctx context.Context, slot phase0.Slot) (phase0.Root, error) {
	resp, err := a.client.(client.AttestationDataProvider).AttestationData(
		ctx,
		&api.AttestationDataOpts{
			Slot:           slot,
			CommitteeIndex: 0,
			Common:         api.CommonOpts{Timeout: 6 * time.Second},
		},
	)
	if err != nil {
		return phase0.Root{}, err
	}

	return resp.Data.BeaconBlockRoot, nil
}

func (a *AttestationMetric) AggregateResults() string {
	missedAttestations := float64(a.missedAttestationsCount.Load())
	missedBlocks := float64(a.missedBlocksCount.Load())
	freshAttestations := float64(a.freshAttestationsCount.Load())
	receivedBlocks := float64(a.receivedBlocksCount.Load())
	unreadyBlocks := float64(a.unreadyBlocksCount.Load())
	correctness := a.LastValue(CorrectnessMeasurement)

	return fmt.Sprintf(
		"missed_attestations=%.0f, unready_blocks_%d_ms=%.0f, missed_blocks=%.0f \n fresh_attestations=%.0f received_blocks=%.0f, correctness=%.2f %%",
		missedAttestations,
		unreadyBlockDelay/time.Millisecond, unreadyBlocks,
		missedBlocks,
		freshAttestations,
		receivedBlocks,
		correctness)
}

func (a *AttestationMetric) calculateMeasurements(slot phase0.Slot) {
	loggerArgs := a.consensusClientLoggerArgs()

	// finalizeSlot consumes both maps' entries for this slot atomically
	// with advancing the watermark, so neither map can gain a new entry for
	// this slot afterwards (see recordHeadEvent/recordAttestationRoot).
	eventBlockRoot, hasEvent, attestationBlockRoot, hasAttestation := a.finalizeSlot(slot)

	if !hasEvent {
		a.missedBlocksCount.Add(1)
		a.AddDataPoint(map[string]float64{
			MissedBlockMeasurement: 1,
		})

		missedBlocksMetric.With(serverAddrLabel(a.client.Address())).Inc()

		logger.WriteMetric(metric.ConsensusGroup, a.Name, map[string]any{
			MissedBlockMeasurement: 1,
		}, loggerArgs)
		return
	}

	a.receivedBlocksCount.Add(1)
	a.AddDataPoint(map[string]float64{
		ReceivedBlockMeasurement: 1,
	})

	receivedBlocksMetric.With(serverAddrLabel(a.client.Address())).Inc()

	logger.WriteMetric(metric.ConsensusGroup, a.Name, map[string]any{
		ReceivedBlockMeasurement: 1,
	}, loggerArgs)

	defer a.calculateCorrectness()

	if !hasAttestation {
		a.missedAttestationsCount.Add(1)
		a.AddDataPoint(map[string]float64{
			MissedAttestationMeasurement: 1,
		})

		missedAttestationsMetric.With(serverAddrLabel(a.client.Address())).Inc()

		logger.WriteMetric(metric.ConsensusGroup, a.Name, map[string]any{
			MissedAttestationMeasurement: 1,
		}, loggerArgs)

		return
	}

	if attestationBlockRoot == eventBlockRoot.RootBlock {
		a.freshAttestationsCount.Add(1)
		a.AddDataPoint(map[string]float64{
			FreshAttestationMeasurement: 1,
		})

		freshAttestationsMetric.With(serverAddrLabel(a.client.Address())).Inc()

		logger.WriteMetric(metric.ConsensusGroup, a.Name, map[string]any{
			FreshAttestationMeasurement: 1,
		}, loggerArgs)
	}
}

func (a *AttestationMetric) calculateCorrectness() {
	freshAttestations := float64(a.freshAttestationsCount.Load())
	receivedBlocks := float64(a.receivedBlocksCount.Load())

	correctness := freshAttestations / receivedBlocks * 100

	a.AddDataPoint(map[string]float64{
		CorrectnessMeasurement: correctness,
	})

	correctnessMetric.With(serverAddrLabel(a.client.Address())).Set(correctness)

	logger.WriteMetric(metric.ConsensusGroup, a.Name, map[string]any{
		CorrectnessMeasurement: correctness,
	}, a.consensusClientLoggerArgs())
}

func slotTime(genesisTime time.Time, slot phase0.Slot) time.Time {
	return genesisTime.Add(time.Duration(slot) * 12 * time.Second)
}

func currentSlot(genesisTime time.Time) phase0.Slot {
	return phase0.Slot(time.Since(genesisTime) / (12 * time.Second))
}

func (a *AttestationMetric) consensusClientLoggerArgs() map[string]any {
	return map[string]any{
		"client_addr":   a.client.Address(),
		"client_active": a.client.IsActive(),
		"client_synced": a.client.IsSynced(),
	}
}
