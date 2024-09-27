package report

import (
	"fmt"
	"os"
	"time"

	"github.com/aquasecurity/table"
)

var consensusHeaders = []string{
	"Consensus Client Response Time: avg",
	"Consensus Client Response Time: > 1sec",
}

type ConsensusRecord struct {
	ConsensusClientResponseTimeAvg     time.Duration
	ConsensusClientResponseTimeDelayed string
}

type ConsensusReport struct {
	t *table.Table
}

func NewConsensus() *ConsensusReport {
	t := table.New(os.Stdout)

	t.SetHeaders(consensusHeaders...)

	var alignments []table.Alignment
	for i := 0; i < len(consensusHeaders); i++ {
		alignments = append(alignments, table.AlignCenter)
	}
	t.SetAlignment(alignments...)

	return &ConsensusReport{
		t: t,
	}
}

func (r *ConsensusReport) AddRecord(record ConsensusRecord) {
	r.t.AddRow(
		fmt.Sprint(record.ConsensusClientResponseTimeAvg),
		record.ConsensusClientResponseTimeDelayed,
	)
}

func (r *ConsensusReport) Render() {
	r.t.Render()
}
