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
	"Score",
	"Commit: Total Delay",
	"Prepare: avg",
	"Prepare: highest",
	"Prepare: > 1sec"}

type OperatorRecord struct {
	OperatorID          uint64
	Score               uint64
	CommitDelayTotal    time.Duration
	PrepareDelayAvg     time.Duration
	PrepareHighestDelay time.Duration
	PrepareMoreThanSec  string
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
	if !r.withScores {
		r.t.AddRow(
			fmt.Sprint(record.OperatorID),
			record.CommitDelayTotal.String(),
			record.PrepareDelayAvg.String(),
			record.PrepareHighestDelay.String(),
			record.PrepareMoreThanSec,
		)
		return
	}
	r.t.AddRow(
		fmt.Sprint(record.OperatorID),
		fmt.Sprint(record.Score),
		record.CommitDelayTotal.String(),
		record.PrepareDelayAvg.String(),
		record.PrepareHighestDelay.String(),
		record.PrepareMoreThanSec,
	)
}

func (r *OperatorReport) Render() {
	r.t.Render()
}
