package report

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/aquasecurity/table"
)

var headers = []string{
	"Operator",
	"Beacon Response Time: avg",
	"Beacon Response Time: > 1sec",
	"Score",
	"Commit: Total Delay",
	"Prepare: avg",
	"Prepare: highest",
	"Prepare: > 1sec"}

type Record struct {
	OperatorID                uint64
	BeaconResponseTimeAvg     time.Duration
	BeaconResponseTimeDelayed string
	Score                     uint64
	CommitDelayTotal          time.Duration
	PrepareDelayAvg           time.Duration
	PrepareHighestDelay       time.Duration
	PrepareMoreThanSec        string
}

type Report struct {
	withScores bool
	t          *table.Table
}

func New(withScores bool) *Report {
	t := table.New(os.Stdout)

	if !withScores {
		index := slices.Index(headers, "Score")
		headers = slices.Delete(headers, index, index+1)
	}

	t.SetHeaders(headers...)
	t.SetAutoMerge(true)

	var alignments []table.Alignment
	for i := 0; i < len(headers); i++ {
		alignments = append(alignments, table.AlignCenter)
	}
	t.SetAlignment(alignments...)

	return &Report{
		withScores: withScores,
		t:          t,
	}
}

func (r *Report) AddRecord(record Record) {
	if !r.withScores {
		r.t.AddRow(
			fmt.Sprint(record.OperatorID),
			fmt.Sprint(record.BeaconResponseTimeAvg),
			record.BeaconResponseTimeDelayed,
			record.CommitDelayTotal.String(),
			record.PrepareDelayAvg.String(),
			record.PrepareHighestDelay.String(),
			record.PrepareMoreThanSec,
		)
		return
	}
	r.t.AddRow(
		fmt.Sprint(record.OperatorID),
		fmt.Sprint(record.BeaconResponseTimeAvg),
		record.BeaconResponseTimeDelayed,
		fmt.Sprint(record.Score),
		record.CommitDelayTotal.String(),
		record.PrepareDelayAvg.String(),
		record.PrepareHighestDelay.String(),
		record.PrepareMoreThanSec,
	)
}

func (r *Report) Render() {
	r.t.Render()
}
