package report

import (
	"fmt"
	"os"
	"time"

	"github.com/aquasecurity/table"
)

var headers = []string{"Operator", "Beacon Time: avg", "Beacon Time: > 1sec", "Score", "Commit: Total Delay", "Prepare: avg", "Prepare: highest", "Prepare: > 1sec"}

type Record struct {
	OperatorID            uint64
	BeaconTimeAvg         time.Duration
	BeaconTimeMoreThanSec string
	Score                 uint64
	CommitDelayTotal      time.Duration
	PrepareDelayAvg       time.Duration
	PrepareHighestDelay   time.Duration
	PrepareMoreThanSec    string
}

type Report struct {
	t *table.Table
}

func New() *Report {
	t := table.New(os.Stdout)

	t.SetHeaders(headers...)

	var alignments []table.Alignment
	for i := 0; i < len(headers); i++ {
		alignments = append(alignments, table.AlignCenter)
	}
	t.SetAlignment(alignments...)

	return &Report{
		t: t,
	}
}

func (r *Report) AddRecord(record Record) {
	r.t.AddRow(
		fmt.Sprint(record.OperatorID),
		fmt.Sprint(record.BeaconTimeAvg),
		record.BeaconTimeMoreThanSec,
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
