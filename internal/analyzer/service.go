package analyzer

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/attestation"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/commit"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/prepare"
)

type (
	attestationService interface {
		Analyze() (attestation.Stats, error)
	}
	commitService interface {
		Analyze() (map[parser.SignerID]commit.Stats, error)
	}

	prepareService interface {
		Analyze() (map[parser.SignerID]prepare.Stats, error)
	}
)

type OperatorResult struct {
	OperatorID uint64

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

type Service struct {
	attestationAnalyzer attestationService
	commitAnalyzer      commitService
	prepareAnalyzer     prepareService
	operators           []uint32
	cluster             bool
}

func New(
	attestationSvc attestationService,
	commitSvc commitService,
	prepareSvc prepareService,
	operators []uint32,
	cluster bool) (*Service, error) {
	return &Service{
		attestationAnalyzer: attestationSvc,
		commitAnalyzer:      commitSvc,
		prepareAnalyzer:     prepareSvc,
		operators:           operators,
		cluster:             cluster,
	}, nil
}

func (r *Service) Start() ([]OperatorResult, error) {
	commitStats, prepareStats, attestationStats, err := r.runAnalyzers()
	if err != nil {
		return nil, err
	}

	ids := collectDistinctIDs(commitStats, prepareStats)

	var result []OperatorResult
	for _, id := range ids {
		commitSignerScore := commitStats[id].Score
		commitTotalDelay := commitStats[id].Delay

		result = append(result, OperatorResult{
			OperatorID: uint64(id),

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

func (r *Service) runAnalyzers() (map[parser.SignerID]commit.Stats, map[parser.SignerID]prepare.Stats, attestation.Stats, error) {
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	var (
		commitStats      map[parser.SignerID]commit.Stats
		prepareStats     map[parser.SignerID]prepare.Stats
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

	wg.Wait()
	close(errChan)

	for e := range errChan {
		if e != nil {
			return commitStats, prepareStats, attestationStats, e
		}
	}
	return commitStats, prepareStats, attestationStats, nil
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
