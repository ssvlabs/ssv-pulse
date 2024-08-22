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
	MissedBlockMeasurement       = "MissedBlock"
	ReceivedBlockMeasurement     = "ReceivedBlock"
	MissedAttestationMeasurement = "MissedAttestation"
	FreshAttestationMeasurement  = "FreshAttestation"
)

type (
	SlotData struct {
		Received  time.Time
		RootBlock phase0.Root
	}

	AttestationMetric struct {
		metric.Base[float64]
		client             client.Service
		genesisTime        time.Time
		receivals          sync.Map
		receivedBlockRoots sync.Map
		isLaunched         bool
		quitChan           chan struct{}
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
		client:             client,
		quitChan:           make(chan struct{}),
		receivals:          sync.Map{},
		receivedBlockRoots: sync.Map{},
		genesisTime:        genesisTime,
	}
}

func (a *AttestationMetric) Measure() {
	if a.isLaunched {
		return
	}

	go a.launchListener()

	time.Sleep(time.Until(slotTime(a.genesisTime, currentSlot(a.genesisTime))))

	slotTicker := time.NewTicker(blockMintingTime)

	go func() {
		for {
			select {
			case <-slotTicker.C:
				go func() {
					a.fetchAttestationData()
					a.calculateMeasurements()
				}()
			case <-a.quitChan:
				slotTicker.Stop()
				return
			}
		}
	}()

	a.isLaunched = true
}

func (a *AttestationMetric) fetchAttestationData() {
	slot := currentSlot(a.genesisTime)
	attestationData, err := a.client.(client.AttestationDataProvider).AttestationData(
		context.TODO(),
		&api.AttestationDataOpts{
			Slot:           slot,
			CommitteeIndex: 0,
			Common:         api.CommonOpts{Timeout: 6 * time.Second},
		},
	)
	if err != nil {
		logger.WriteError(metric.ConsensusGroup, a.Name, err)
		return
	}

	a.receivedBlockRoots.Store(slot, attestationData.Data.BeaconBlockRoot)
}

func (a *AttestationMetric) launchListener() {
	if err := a.client.(client.EventsProvider).Events(
		context.TODO(),
		[]string{"head"},
		func(event *v1.Event) {
			data := event.Data.(*v1.HeadEvent)
			a.receivals.Store(data.Slot, SlotData{
				Received:  time.Now(),
				RootBlock: data.Block,
			})
		},
	); err != nil {
		logger.WriteError(metric.ConsensusGroup, a.Name, err)
	}
}

func (a *AttestationMetric) AggregateResults() string {
	close(a.quitChan)
	var missedAttestations, freshAttestations, missedBlocks, receivedBlocks float64

	for _, point := range a.DataPoints {
		missedAttestations += point.Values[MissedAttestationMeasurement]
		missedBlocks += point.Values[MissedBlockMeasurement]
		freshAttestations += point.Values[FreshAttestationMeasurement]
		receivedBlocks += point.Values[ReceivedBlockMeasurement]
	}

	return fmt.Sprintf("missed_attestations=%.0f, fresh_attestations=%.0f, missed_blocks=%.0f \n received_blocks=%.0f, correctness=%.2f %%",
		missedAttestations,
		freshAttestations,
		missedBlocks,
		receivedBlocks,
		freshAttestations/receivedBlocks*100)
}

func (a *AttestationMetric) calculateMeasurements() {
	currentSlot := currentSlot(a.genesisTime)
	receival, ok := a.receivals.Load(currentSlot)

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

	blockRoot, ok := a.receivedBlockRoots.Load(currentSlot)
	if !ok {
		a.AddDataPoint(map[string]float64{
			MissedAttestationMeasurement: 1,
		})

		logger.WriteMetric(metric.ConsensusGroup, a.Name, map[string]any{
			MissedAttestationMeasurement: 1,
		})
		return
	}

	if blockRoot == receival.(SlotData).RootBlock {
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
