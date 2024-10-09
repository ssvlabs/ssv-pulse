package analyzer

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/attestation"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/commit"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/operator"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/prepare"
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
	}

	AnalyzerResult struct {
		OperatorStats []OperatorStats
	}
)

type Service struct {
	operatorAnalyzer    operatorAnalyzer
	attestationAnalyzer attestationAnalyzer
	commitAnalyzer      commitAnalyzer
	prepareAnalyzer     prepareAnalyzer
	operators           []uint32
	cluster             bool
}

func New(
	operatorAnalyzer operatorAnalyzer,
	attestationSvc attestationAnalyzer,
	commitSvc commitAnalyzer,
	prepareSvc prepareAnalyzer,
	operators []uint32,
	cluster bool) (*Service, error) {

	return &Service{
		operatorAnalyzer:    operatorAnalyzer,
		attestationAnalyzer: attestationSvc,
		commitAnalyzer:      commitSvc,
		prepareAnalyzer:     prepareSvc,
		operators:           operators,
		cluster:             cluster,
	}, nil
}

func (r *Service) Start() (AnalyzerResult, error) {
	var result AnalyzerResult
	operatorStats, commitStats, prepareStats, attestationStats, err := r.runAnalyzers()
	if err != nil {
		return result, err
	}

	ids := collectDistinctIDs(commitStats, prepareStats)

	for _, id := range ids {
		commitSignerScore := commitStats[id].Score
		commitTotalDelay := commitStats[id].Delay

		result.OperatorStats = append(result.OperatorStats, OperatorStats{
			OperatorID:     uint64(id),
			IsLogFileOwner: uint64(id) == uint64(operatorStats.Owner),

			AttestationTimeAverage: attestationStats.AttestationTimeTotal / time.Duration(len(attestationStats.AttestationDurations)),
			AttestationTimeCount:   uint16(len(attestationStats.AttestationDurations)),
			AttestationDelayCount:  attestationStats.AttestationDelayCount,

			CommitSignerScore: uint64(commitSignerScore),
			CommitTotalDelay:  commitTotalDelay,

			PrepareDelayAvg:     prepareStats[id].AverageDelay,
			PrepareHighestDelay: prepareStats[id].HighestDelay,
			PrepareDelayCount:   prepareStats[id].MoreSecondDelay,
			PrepareCount:        prepareStats[id].Count,
		})
	}

	return result, nil
}

func (r *Service) runAnalyzers() (operator.Stats, map[parser.SignerID]commit.Stats, map[parser.SignerID]prepare.Stats, attestation.Stats, error) {
	var wg sync.WaitGroup
	errChan := make(chan error, 4)

	var (
		commitStats      map[parser.SignerID]commit.Stats
		prepareStats     map[parser.SignerID]prepare.Stats
		operatorStats    operator.Stats
		attestationStats attestation.Stats
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

	wg.Wait()
	close(errChan)

	for e := range errChan {
		if e != nil {
			return operatorStats, commitStats, prepareStats, attestationStats, e
		}
	}
	return operatorStats, commitStats, prepareStats, attestationStats, nil
}

func collectDistinctIDs(commitStats map[parser.SignerID]commit.Stats, proposeStats map[parser.SignerID]prepare.Stats) []parser.SignerID {
	tmpIDs := make(map[parser.SignerID]bool)

	for singerID := range commitStats {
		tmpIDs[singerID] = true
	}

	for signerID := range proposeStats {
		if _, exist := tmpIDs[signerID]; !exist {
			tmpIDs[signerID] = true
		}
	}

	return slices.Collect(maps.Keys(tmpIDs))
}
