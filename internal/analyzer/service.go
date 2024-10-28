package analyzer

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/client"
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
	clientAnalyzer interface {
		Analyze() (client.Stats, error)
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
		CommitDelayPercent map[time.Duration]float32
		CommitCount        uint16

		ConsensusClientResponseTimeAvg,
		ConsensusClientResponseTimeP10,
		ConsensusClientResponseTimeP50,
		ConsensusClientResponseTimeP90 time.Duration
		ConsensusClientResponseTimeDelayPercent map[time.Duration]float32

		SSVClientCrashesTotal,
		SSVClientELUnhealthy,
		SSVClientCLUnhealthy uint16

		PrepareDelayAvg,
		PrepareDelayHighest time.Duration
		PrepareDelayPercent map[time.Duration]float32
		PrepareCount        uint16

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
	peersAnalyzer     peersAnalyzer
	consensusAnalyzer consensusAnalyzer
	operatorAnalyzer  operatorAnalyzer
	clientAnalyzer    clientAnalyzer
	commitAnalyzer    commitAnalyzer
	prepareAnalyzer   prepareAnalyzer
	operators         []uint32
	cluster           bool
}

func New(
	peersAnalyzer peersAnalyzer,
	consensusAnalyzer consensusAnalyzer,
	operatorAnalyzer operatorAnalyzer,
	clientAnalyzer clientAnalyzer,
	commitAnalyzer commitAnalyzer,
	prepareAnalyzer prepareAnalyzer,
	operators []uint32,
	cluster bool) (*Service, error) {

	return &Service{
		peersAnalyzer:     peersAnalyzer,
		consensusAnalyzer: consensusAnalyzer,
		operatorAnalyzer:  operatorAnalyzer,
		clientAnalyzer:    clientAnalyzer,
		commitAnalyzer:    commitAnalyzer,
		prepareAnalyzer:   prepareAnalyzer,
		operators:         operators,
		cluster:           cluster,
	}, nil
}

func (s *Service) Start() (AnalyzerResult, error) {
	var result AnalyzerResult
	peerStats, operatorStats, consensusStats, commitStats, prepareStats, clientStats, err := s.runAnalyzers()
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
			CommitDelayPercent: commitStats[operatorID].DelayedPercent,
			CommitCount:        commitStats[operatorID].Count,

			PrepareDelayAvg:     prepareStats[operatorID].DelayAvg,
			PrepareDelayHighest: prepareStats[operatorID].DelayHighest,
			PrepareDelayPercent: prepareStats[operatorID].DelayedPercent,
			PrepareCount:        prepareStats[operatorID].Count,

			ConsensusTimeAvg: consensusStats[operatorID].ConsensusTimeAvg,
		}

		//these metrics are only available for the log file owner
		if isOwner {
			opStats.Clusters = operatorStats.Clusters[operatorID]

			opStats.ConsensusClientResponseTimeAvg = clientStats.ConsensusClientResponseTimeAvg
			opStats.ConsensusClientResponseTimeDelayPercent = clientStats.ConsensusClientResponseTimeDelayPercent
			opStats.ConsensusClientResponseTimeP10 = clientStats.ConsensusClientResponseTimeP10
			opStats.ConsensusClientResponseTimeP50 = clientStats.ConsensusClientResponseTimeP50
			opStats.ConsensusClientResponseTimeP90 = clientStats.ConsensusClientResponseTimeP90
			opStats.SSVClientCrashesTotal = clientStats.SSVClientCrashesTotal
			opStats.SSVClientCLUnhealthy = clientStats.SSVClientCLUnhealthy
			opStats.SSVClientELUnhealthy = clientStats.SSVClientELUnhealthy

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
	client.Stats,
	error) {
	var wg sync.WaitGroup
	errChan := make(chan error, 6)

	var (
		commitStats    map[parser.SignerID]commit.Stats
		prepareStats   map[parser.SignerID]prepare.Stats
		consensusStats map[parser.SignerID]consensus.Stats
		clientStats    client.Stats
		operatorStats  operator.Stats
		peersStats     peers.Stats
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
		clientStats, err = r.clientAnalyzer.Analyze()
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
			return peersStats, operatorStats, consensusStats, commitStats, prepareStats, clientStats, e
		}
	}

	return peersStats, operatorStats, consensusStats, commitStats, prepareStats, clientStats, nil
}
