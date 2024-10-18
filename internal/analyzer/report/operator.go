package report

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aquasecurity/table"
)

var operatorHeaders = []string{
	"Operator",
	"Clusters",

	"Commit: \n delay avg",
	"Commit: \n delay highest",
	"Commit: \n delayed",
	"Commit: \n total count",

	"Prepare: \n delay avg",
	"Prepare: \n delay highest",
	"Prepare: \n delayed",
	"Prepare: \n total count",
	"Consensus: \n avg",
}

type OperatorRecord struct {
	OperatorID     uint32
	Clusters       [][]uint32
	IsLogFileOwner bool

	CommitDelayAvg,
	CommitDelayHighest time.Duration
	CommitDelayPercent map[time.Duration]float32
	CommitTotalCount   uint16

	PrepareDelayAvg,
	PrepareDelayHighest time.Duration
	PrepareDelayPercent map[time.Duration]float32
	PrepareTotalCount   uint16

	ConsensusTimeAvg time.Duration
}

type OperatorReport struct {
	t *table.Table
}

func NewOperator() *OperatorReport {
	t := table.New(os.Stdout)

	t.SetHeaders("Validator Performance")
	t.AddHeaders(operatorHeaders...)
	t.SetAutoMerge(true)
	t.SetHeaderColSpans(0, len(operatorHeaders))

	var alignments []table.Alignment
	for i := 0; i < len(operatorHeaders); i++ {
		alignments = append(alignments, table.AlignCenter)
	}
	t.SetAlignment(alignments...)

	return &OperatorReport{
		t: t,
	}
}

func (r *OperatorReport) AddRecord(record OperatorRecord) {
	var (
		clusterReportItem string
		operatorID        string = fmt.Sprint(record.OperatorID)
		delayedPrepare    []string
		delayedCommit     []string
	)

	if record.IsLogFileOwner {
		operatorID = fmt.Sprintf("%d ⭐️", record.OperatorID)
		clusterReportItem = fmt.Sprint(record.Clusters)
	}

	for duration, value := range record.PrepareDelayPercent {
		delayedPrepare = append(delayedCommit, fmt.Sprintf("%s: %.2f%% \n", duration.String(), value))
	}

	for duration, value := range record.CommitDelayPercent {
		delayedCommit = append(delayedCommit, fmt.Sprintf("%s: %.2f%% \n", duration.String(), value))
	}

	r.t.AddRow(
		operatorID,
		clusterReportItem,
		record.CommitDelayAvg.String(),
		record.CommitDelayHighest.String(),
		strings.Join(delayedCommit, "\n"),
		fmt.Sprint(record.CommitTotalCount),
		record.PrepareDelayAvg.String(),
		record.PrepareDelayHighest.String(),
		strings.Join(delayedPrepare, "\n"),
		fmt.Sprint(record.PrepareTotalCount),
		record.ConsensusTimeAvg.String(),
	)
}

func (r *OperatorReport) Render() {
	r.t.Render()
}
