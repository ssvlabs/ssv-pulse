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
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/prepare"
	"github.com/ssvlabs/ssv-pulse/internal/utils"
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
		Analyze() (consensus.Stats, error)
	}

	OperatorStats struct {
		OperatorID     uint64
		IsLogFileOwner bool

		CommitSignerScore uint64
		CommitTotalDelay  time.Duration

		AttestationTimeAverage time.Duration
		AttestationTimeCount,
		AttestationDelayCount uint16

		PrepareDelayAvg,
		PrepareHighestDelay time.Duration
		PrepareDelayCount,
		PrepareCount uint16

		ConsensusTimeAvg time.Duration
		ConsensusParticipationCount,
		ConsensusSuccessfulAttestationSubmissions uint16
	}

	AnalyzerResult struct {
		OperatorStats []OperatorStats
	}
)

type Service struct {
	consensusAnalyzer   consensusAnalyzer
	operatorAnalyzer    operatorAnalyzer
	attestationAnalyzer attestationAnalyzer
	commitAnalyzer      commitAnalyzer
	prepareAnalyzer     prepareAnalyzer
	operators           []uint32
	cluster             bool
}

func New(
	consensusAnalyzer consensusAnalyzer,
	operatorAnalyzer operatorAnalyzer,
	attestationAnalyzer attestationAnalyzer,
	commitAnalyzer commitAnalyzer,
	prepareAnalyzer prepareAnalyzer,
	operators []uint32,
	cluster bool) (*Service, error) {

	return &Service{
		consensusAnalyzer:   consensusAnalyzer,
		operatorAnalyzer:    operatorAnalyzer,
		attestationAnalyzer: attestationAnalyzer,
		commitAnalyzer:      commitAnalyzer,
		prepareAnalyzer:     prepareAnalyzer,
		operators:           operators,
		cluster:             cluster,
	}, nil
}

func (r *Service) Start() (AnalyzerResult, error) {
	var result AnalyzerResult
	consensusStats, operatorStats, commitStats, prepareStats, attestationStats, err := r.runAnalyzers()
	if err != nil {
		return result, err
	}

	operatorIDs := utils.CollectDistinct(
		slices.Collect(maps.Keys(commitStats)),
		slices.Collect(maps.Keys(prepareStats)),
		slices.Collect(maps.Keys(consensusStats.OperatorConsensusTimes)),
	)

	for _, operatorID := range operatorIDs {
		commitSignerScore := commitStats[operatorID].Score
		commitTotalDelay := commitStats[operatorID].Delay

		consensusDurations := consensusStats.OperatorConsensusTimes[operatorID]
		var (
			consensusDurationsTotal, consensusDurationAvg time.Duration
			consensusDurationLen                          int = len(consensusDurations)
		)
		for _, duration := range consensusDurations {
			consensusDurationsTotal += duration
		}
		if consensusDurationLen > 0 {
			consensusDurationAvg = consensusDurationsTotal / time.Duration(consensusDurationLen)
		}

		result.OperatorStats = append(result.OperatorStats, OperatorStats{
			OperatorID:     uint64(operatorID),
			IsLogFileOwner: uint64(operatorID) == uint64(operatorStats.Owner),

			AttestationTimeAverage: attestationStats.AttestationTimeTotal / time.Duration(len(attestationStats.AttestationDurations)),
			AttestationTimeCount:   uint16(len(attestationStats.AttestationDurations)),
			AttestationDelayCount:  attestationStats.AttestationDelayCount,

			CommitSignerScore: uint64(commitSignerScore),
			CommitTotalDelay:  commitTotalDelay,

			PrepareDelayAvg:     prepareStats[operatorID].AverageDelay,
			PrepareHighestDelay: prepareStats[operatorID].HighestDelay,
			PrepareDelayCount:   prepareStats[operatorID].MoreSecondDelay,
			PrepareCount:        prepareStats[operatorID].Count,

			ConsensusTimeAvg: consensusDurationAvg,
			ConsensusSuccessfulAttestationSubmissions: consensusStats.SuccessfullySubmittedAttestations,
			ConsensusParticipationCount:               consensusStats.OperatorConsensusParticipation[operatorID],
		})
	}

	return result, nil
}

func (r *Service) runAnalyzers() (consensus.Stats, operator.Stats, map[parser.SignerID]commit.Stats, map[parser.SignerID]prepare.Stats, attestation.Stats, error) {
	var wg sync.WaitGroup
	errChan := make(chan error, 5)

	var (
		commitStats      map[parser.SignerID]commit.Stats
		prepareStats     map[parser.SignerID]prepare.Stats
		attestationStats attestation.Stats
		operatorStats    operator.Stats
		consensusStats   consensus.Stats
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

	wg.Wait()
	close(errChan)

	for e := range errChan {
		if e != nil {
			return consensusStats, operatorStats, commitStats, prepareStats, attestationStats, e
		}
	}
	return consensusStats, operatorStats, commitStats, prepareStats, attestationStats, nil
}
