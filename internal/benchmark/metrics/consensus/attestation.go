package consensus

import (
	"context"
	"fmt"
	"sync"
	"time"

	client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/auto"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/rs/zerolog"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	blockMintingTime             = time.Second * 12
	unreadyBlockDelayMs          = 200
	MissedBlockMeasurement       = "MissedBlock"
	ReceivedBlockMeasurement     = "ReceivedBlock"
	MissedAttestationMeasurement = "MissedAttestation"
	FreshAttestationMeasurement  = "FreshAttestation"
)

var (
	UnreadyBlockMeasurement = fmt.Sprintf("UnreadyBlockMeasurement%dms", unreadyBlockDelayMs)
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
		isLaunched            bool
		quitChan              chan struct{}
	}
)

func NewAttestationMetric(url, name string, genesisTime time.Time, healthCondition []metric.HealthCondition[float64]) *AttestationMetric {
	client, err := auto.New(
		context.TODO(),
		auto.WithLogLevel(zerolog.DebugLevel),
		auto.WithAddress(url),
	)
	if err != nil {
		panic(err.Error())
	}
	return &AttestationMetric{
		Base: metric.Base[float64]{
			HealthConditions: healthCondition,
			Name:             name,
		},
		client:                client,
		quitChan:              make(chan struct{}),
		eventBlockRoots:       sync.Map{},
		attestationBlockRoots: sync.Map{},
		genesisTime:           genesisTime,
	}
}

func (a *AttestationMetric) Measure() {
	if a.isLaunched {
		return
	}
	ctx := context.TODO()

	go a.launchListener(ctx)

	go func() {
		slot := currentSlot(a.genesisTime)
		const calculationSlotLag = 2
		laggedSlot := slot + calculationSlotLag
		for {
			slot++
			nextSlotWithDelay := time.After(time.Until(slotTime(a.genesisTime, slot).Add(time.Second * 4)))
			select {
			case <-nextSlotWithDelay:
				go func(slot phase0.Slot) {
					a.fetchAttestationData(ctx, slot)

					if slot > laggedSlot {
						a.calculateMeasurements(slot - calculationSlotLag)
					}
				}(slot)
			case <-a.quitChan:
				return
			}
		}
	}()

	a.isLaunched = true
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
	if err := a.client.(client.EventsProvider).Events(
		ctx,
		[]string{"head"},
		func(event *v1.Event) {
			data := event.Data.(*v1.HeadEvent)

			a.eventBlockRoots.Store(data.Slot, SlotData{
				Received:  time.Now(),
				RootBlock: data.Block,
			})

			go a.checkUnreadyBlock(data.Slot, data.Block)
		},
	); err != nil {
		logger.WriteError(metric.ConsensusGroup, a.Name, err)
	}
}

func (a *AttestationMetric) checkUnreadyBlock(slot phase0.Slot, block phase0.Root) {
	time.Sleep(time.Millisecond * unreadyBlockDelayMs)
	blockRoot, err := a.fetchAttestationBlockRoot(context.TODO(), slot)
	if err != nil {
		logger.WriteError(metric.ConsensusGroup, a.Name, err)
		return
	}

	if blockRoot != block {
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
	close(a.quitChan)
	var missedAttestations, freshAttestations, missedBlocks, receivedBlocks, unreadyBlocks float64

	for _, point := range a.DataPoints {
		missedAttestations += point.Values[MissedAttestationMeasurement]
		missedBlocks += point.Values[MissedBlockMeasurement]
		freshAttestations += point.Values[FreshAttestationMeasurement]
		receivedBlocks += point.Values[ReceivedBlockMeasurement]
		unreadyBlocks += point.Values[UnreadyBlockMeasurement]
	}

	return fmt.Sprintf(
		"missed_attestations=%.0f, unready_blocks_%d_ms=%.0f, missed_blocks=%.0f \n fresh_attestations=%.0f received_blocks=%.0f, correctness=%.2f %%",
		missedAttestations,
		unreadyBlockDelayMs, unreadyBlocks,
		missedBlocks,
		freshAttestations,
		receivedBlocks,
		freshAttestations/receivedBlocks*100)
}

func (a *AttestationMetric) calculateMeasurements(slot phase0.Slot) {
	eventBlockRoot, ok := a.eventBlockRoots.Load(slot)

	if !ok {
		a.AddDataPoint(map[string]float64{
			MissedBlockMeasurement: 1,
		})

		logger.WriteMetric(metric.ConsensusGroup, a.Name, map[string]any{
			MissedBlockMeasurement: 1,
		})
		return
	}

	a.AddDataPoint(map[string]float64{
		ReceivedBlockMeasurement: 1,
	})
	logger.WriteMetric(metric.ConsensusGroup, a.Name, map[string]any{
		ReceivedBlockMeasurement: 1,
	})

	attestationBlockRoot, ok := a.attestationBlockRoots.Load(slot)
	if !ok {
		a.AddDataPoint(map[string]float64{
			MissedAttestationMeasurement: 1,
		})

		logger.WriteMetric(metric.ConsensusGroup, a.Name, map[string]any{
			MissedAttestationMeasurement: 1,
		})
		return
	}

	if attestationBlockRoot == eventBlockRoot.(SlotData).RootBlock {
		a.AddDataPoint(map[string]float64{
			FreshAttestationMeasurement: 1,
		})

		logger.WriteMetric(metric.ConsensusGroup, a.Name, map[string]any{
			FreshAttestationMeasurement: 1,
		})
	}
}

func slotTime(genesisTime time.Time, slot phase0.Slot) time.Time {
	return genesisTime.Add(time.Duration(slot) * 12 * time.Second)
}

func currentSlot(genesisTime time.Time) phase0.Slot {
	return phase0.Slot(time.Since(genesisTime) / (12 * time.Second))
}
