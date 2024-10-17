package analyzer

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/attestation"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/commit"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/consensus"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/operator"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/peers"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/prepare"
	"github.com/ssvlabs/ssv-pulse/internal/platform/array"
)

type (
	operatorAnalyzer interface {
		Analyze() (operator.Stats, error)
	}
	attestationAnalyzer interface {
		Analyze() (attestation.Stats, error)
	}
	commitAnalyzer interface {
		Analyze() (map[parser.SignerID]commit.Stats, error)
	}

	prepareAnalyzer interface {
		Analyze() (map[parser.SignerID]prepare.Stats, error)
	}

	consensusAnalyzer interface {
		Analyze() (map[parser.SignerID]consensus.Stats, error)
	}

	peersAnalyzer interface {
		Analyze() (peers.Stats, error)
	}

	OperatorStats struct {
		OperatorID     uint32
		IsLogFileOwner bool
		Clusters       [][]uint32

		CommitTotalDelay,
		CommitDelayAvg,
		CommitDelayHighest time.Duration
		CommitDelayCount map[time.Duration]uint16
		CommitCount      uint16

		ConsensusClientResponseTimeAvg        time.Duration
		ConsensusClientResponseTimeDelayCount map[time.Duration]uint16

		PrepareDelayAvg,
		PrepareDelayHighest time.Duration
		PrepareDelayCount map[time.Duration]uint16
		PrepareCount      uint16

		ConsensusTimeAvg time.Duration

		PeersCountAvg         parser.Metric[float64]
		PeerSSVClientVersions []string
		PeerID                string
	}

	AnalyzerResult struct {
		OperatorStats []OperatorStats
	}
)

type Service struct {
	peersAnalyzer       peersAnalyzer
	consensusAnalyzer   consensusAnalyzer
	operatorAnalyzer    operatorAnalyzer
	attestationAnalyzer attestationAnalyzer
	commitAnalyzer      commitAnalyzer
	prepareAnalyzer     prepareAnalyzer
	operators           []uint32
	cluster             bool
}

func New(
	peersAnalyzer peersAnalyzer,
	consensusAnalyzer consensusAnalyzer,
	operatorAnalyzer operatorAnalyzer,
	attestationAnalyzer attestationAnalyzer,
	commitAnalyzer commitAnalyzer,
	prepareAnalyzer prepareAnalyzer,
	operators []uint32,
	cluster bool) (*Service, error) {

	return &Service{
		peersAnalyzer:       peersAnalyzer,
		consensusAnalyzer:   consensusAnalyzer,
		operatorAnalyzer:    operatorAnalyzer,
		attestationAnalyzer: attestationAnalyzer,
		commitAnalyzer:      commitAnalyzer,
		prepareAnalyzer:     prepareAnalyzer,
		operators:           operators,
		cluster:             cluster,
	}, nil
}

func (s *Service) Start() (AnalyzerResult, error) {
	var result AnalyzerResult
	peerStats, operatorStats, consensusStats, commitStats, prepareStats, attestationStats, err := s.runAnalyzers()
	if err != nil {
		return result, err
	}

	operatorIDs := array.CollectDistinct(
		slices.Collect(maps.Keys(commitStats)),
		slices.Collect(maps.Keys(prepareStats)),
		slices.Collect(maps.Keys(consensusStats)),
	)

	for _, operatorID := range operatorIDs {
		if s.cluster || len(s.operators) != 0 {
			isSupportedOperator := slices.Contains(s.operators, operatorID)
			if !isSupportedOperator {
				continue
			}
		}

		isOwner := operatorID == operatorStats.Owner

		opStats := OperatorStats{
			OperatorID:     operatorID,
			IsLogFileOwner: isOwner,

			CommitTotalDelay:   commitStats[operatorID].DelayTotal,
			CommitDelayAvg:     commitStats[operatorID].DelayAvg,
			CommitDelayHighest: commitStats[operatorID].DelayHighest,
			CommitDelayCount:   commitStats[operatorID].Delayed,
			CommitCount:        commitStats[operatorID].Count,

			PrepareDelayAvg:     prepareStats[operatorID].DelayAvg,
			PrepareDelayHighest: prepareStats[operatorID].DelayHighest,
			PrepareDelayCount:   prepareStats[operatorID].Delayed,
			PrepareCount:        prepareStats[operatorID].Count,

			ConsensusTimeAvg: consensusStats[operatorID].ConsensusTimeAvg,
		}

		//these metrics are only available for the log file owner
		if isOwner {
			opStats.Clusters = operatorStats.Clusters[operatorID]
			opStats.ConsensusClientResponseTimeAvg = attestationStats.ConsensusClientResponseTimeTotal / time.Duration(len(attestationStats.ConsensusClientResponseDurations))
			opStats.ConsensusClientResponseTimeDelayCount = attestationStats.ConsensusClientResponseTimeDelayCount
			opStats.PeersCountAvg = peerStats.PeerCountAvg
			opStats.PeerSSVClientVersions = peerStats.PeerSSVClientVersions
			opStats.PeerID = peerStats.PeerID
		}

		result.OperatorStats = append(result.OperatorStats, opStats)
	}

	return result, nil
}

func (r *Service) runAnalyzers() (
	peers.Stats,
	operator.Stats,
	map[parser.SignerID]consensus.Stats,
	map[parser.SignerID]commit.Stats,
	map[parser.SignerID]prepare.Stats,
	attestation.Stats,
	error) {
	var wg sync.WaitGroup
	errChan := make(chan error, 6)

	var (
		commitStats      map[parser.SignerID]commit.Stats
		prepareStats     map[parser.SignerID]prepare.Stats
		consensusStats   map[parser.SignerID]consensus.Stats
		attestationStats attestation.Stats
		operatorStats    operator.Stats
		peersStats       peers.Stats
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		commitStats, err = r.commitAnalyzer.Analyze()
		if err != nil {
			errChan <- err
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		prepareStats, err = r.prepareAnalyzer.Analyze()
		if err != nil {
			errChan <- err
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		attestationStats, err = r.attestationAnalyzer.Analyze()
		if err != nil {
			errChan <- err
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		operatorStats, err = r.operatorAnalyzer.Analyze()
		if err != nil {
			errChan <- err
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		consensusStats, err = r.consensusAnalyzer.Analyze()
		if err != nil {
			errChan <- err
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		peersStats, err = r.peersAnalyzer.Analyze()
		if err != nil {
			errChan <- err
			return
		}
	}()

	wg.Wait()
	close(errChan)

	for e := range errChan {
		if e != nil {
			return peersStats, operatorStats, consensusStats, commitStats, prepareStats, attestationStats, e
		}
	}

	return peersStats, operatorStats, consensusStats, commitStats, prepareStats, attestationStats, nil
}
