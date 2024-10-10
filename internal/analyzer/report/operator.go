package report

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/aquasecurity/table"
)

var operatorHeaders = []string{
	"Operator",
	"You",
	"Score",
	"Commit: Total Delay",
	"Prepare: avg",
	"Prepare: highest",
	"Prepare: > 1sec",
	"Consensus: avg",
	"Consensus: \n operator participation",
	"Consensus: \n successful attestation submissions",
}

type OperatorRecord struct {
	OperatorID       uint64
	IsLogFileOwner   bool
	Score            uint64
	CommitDelayTotal time.Duration

	PrepareDelayAvg     time.Duration
	PrepareHighestDelay time.Duration
	PrepareMoreThanSec  string

	ConsensusTimeAvg time.Duration
	ConsensusSuccessfulAttestationSubmissions,
	ConsensusParticipation uint16
}

type OperatorReport struct {
	withScores bool
	t          *table.Table
}

func NewOperator(withScores bool) *OperatorReport {
	t := table.New(os.Stdout)

	if !withScores {
		index := slices.Index(operatorHeaders, "Score")
		operatorHeaders = slices.Delete(operatorHeaders, index, index+1)
	}

	t.SetHeaders(operatorHeaders...)
	t.SetAutoMerge(true)

	var alignments []table.Alignment
	for i := 0; i < len(operatorHeaders); i++ {
		alignments = append(alignments, table.AlignCenter)
	}
	t.SetAlignment(alignments...)

	return &OperatorReport{
		withScores: withScores,
		t:          t,
	}
}

func (r *OperatorReport) AddRecord(record OperatorRecord) {
	var ownerSign string
	if record.IsLogFileOwner {
		ownerSign = "⭐️"
	}

	if !r.withScores {
		r.t.AddRow(
			fmt.Sprint(record.OperatorID),
			ownerSign,
			record.CommitDelayTotal.String(),
			record.PrepareDelayAvg.String(),
			record.PrepareHighestDelay.String(),
			record.PrepareMoreThanSec,
			record.ConsensusTimeAvg.String(),
			fmt.Sprint(record.ConsensusParticipation),
			fmt.Sprint(record.ConsensusSuccessfulAttestationSubmissions),
		)
		return
	}
	r.t.AddRow(
		fmt.Sprint(record.OperatorID),
		ownerSign,
		fmt.Sprint(record.Score),
		record.CommitDelayTotal.String(),
		record.PrepareDelayAvg.String(),
		record.PrepareHighestDelay.String(),
		record.PrepareMoreThanSec,
		record.ConsensusTimeAvg.String(),
		fmt.Sprint(record.ConsensusParticipation),
		fmt.Sprint(record.ConsensusSuccessfulAttestationSubmissions),
	)
}

func (r *OperatorReport) Render() {
	r.t.Render()
}
