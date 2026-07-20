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
		client                client.Service
		genesisTime           time.Time
		eventBlockRoots       sync.Map
		attestationBlockRoots sync.Map

		// Running counters backing AggregateResults, updated at the same
		// points AddDataPoint is called below. Kept separately from
		// Base[T] because Base no longer retains full history.
		missedBlocksCount       atomic.Uint64
		receivedBlocksCount     atomic.Uint64
		missedAttestationsCount atomic.Uint64
		freshAttestationsCount  atomic.Uint64
		unreadyBlocksCount      atomic.Uint64

		// Highest slot calculateMeasurements has finalized. Used to reject
		// head events that arrive after their slot was already processed
		// (and will never be revisited), so they don't leak a permanent
		// entry into eventBlockRoots.
		lastProcessedSlot atomic.Uint64
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
		eventBlockRoots:       sync.Map{},
		attestationBlockRoots: sync.Map{},
		genesisTime:           genesisTime,
	}
}

func (a *AttestationMetric) Measure(ctx context.Context) {
	go a.launchListener(ctx)

	go func() {
		genesisSlot := currentSlot(a.genesisTime)
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

	a.attestationBlockRoots.Store(slot, blockRoot)
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
// unless calculateMeasurements has already finalized that slot — a late
// event for an already-processed slot would never be consumed and would
// otherwise leak a permanent entry into eventBlockRoots.
func (a *AttestationMetric) recordHeadEvent(slot phase0.Slot, block phase0.Root) {
	if uint64(slot) <= a.lastProcessedSlot.Load() {
		return
	}

	a.eventBlockRoots.Store(slot, SlotData{
		Received:  time.Now(),
		RootBlock: block,
	})
}

// advanceLastProcessedSlot moves lastProcessedSlot forward to slot, unless
// it has already advanced past it. Per-slot goroutines can finish out of
// order (fetchAttestationData has variable network latency), so this uses a
// compare-and-swap loop rather than an unconditional store to guard against
// a late, out-of-order call moving the watermark backwards.
func (a *AttestationMetric) advanceLastProcessedSlot(slot phase0.Slot) {
	for {
		current := a.lastProcessedSlot.Load()
		if uint64(slot) <= current {
			return
		}
		if a.lastProcessedSlot.CompareAndSwap(current, uint64(slot)) {
			return
		}
	}
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

	// Finalize this slot before touching either map: any head event for
	// this slot or earlier that arrives from here on is too late to matter
	// and must not be stored (see recordHeadEvent).
	a.advanceLastProcessedSlot(slot)

	// LoadAndDelete: each slot's entry is only ever needed for this one
	// lookup, so consuming it here keeps both maps bounded to in-flight
	// slots instead of growing for the lifetime of the process.
	eventBlockRoot, ok := a.eventBlockRoots.LoadAndDelete(slot)
	if !ok {
		// fetchAttestationData may have already written this slot's
		// attestation root before we learned the block itself was missed —
		// discard it so it doesn't leak, since it will never be read now.
		a.attestationBlockRoots.Delete(slot)

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

	attestationBlockRoot, ok := a.attestationBlockRoots.LoadAndDelete(slot)
	if !ok {
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

	if attestationBlockRoot == eventBlockRoot.(SlotData).RootBlock {
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
