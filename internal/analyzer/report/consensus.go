package report

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aquasecurity/table"
)

var consensusHeaders = []string{
	"Operator",
	"Consensus Client Response Time: avg",
	"Consensus Client Response Time: delayed",
}

type ConsensusRecord struct {
	OperatorID                            uint32
	ConsensusClientResponseTimeAvg        time.Duration
	ConsensusClientResponseTimeDelayCount map[time.Duration]uint16
}

type ConsensusReport struct {
	records []ConsensusRecord
	t       *table.Table
}

func NewConsensus() *ConsensusReport {
	t := table.New(os.Stdout)

	t.SetHeaders("Consensus Client Performance")
	t.AddHeaders(consensusHeaders...)
	t.SetAutoMerge(true)
	t.SetHeaderColSpans(0, len(consensusHeaders))

	var alignments []table.Alignment
	for i := 0; i < len(consensusHeaders); i++ {
		alignments = append(alignments, table.AlignCenter)
	}
	t.SetAlignment(alignments...)

	return &ConsensusReport{
		t: t,
	}
}

func (c *ConsensusReport) AddRecord(record ConsensusRecord) {
	c.records = append(c.records, record)
}

func (c *ConsensusReport) Render() {
	type consensusRecordAggregate struct {
		delayedResponses                       map[time.Duration]uint16
		consensusClientResponseTimeAvgTotal    time.Duration
		consensusClientResponseTimeRecordCount uint16
	}

	consensusRecordAggregates := make(map[uint32]consensusRecordAggregate)

	for _, record := range c.records {
		aggregate := consensusRecordAggregates[record.OperatorID]

		for duration, value := range record.ConsensusClientResponseTimeDelayCount {
			_, ok := aggregate.delayedResponses[duration]
			if !ok {
				aggregate.delayedResponses = make(map[time.Duration]uint16)
			}
			aggregate.delayedResponses[duration] += value
		}

		aggregate.consensusClientResponseTimeAvgTotal += record.ConsensusClientResponseTimeAvg
		aggregate.consensusClientResponseTimeRecordCount++

		consensusRecordAggregates[record.OperatorID] = aggregate
	}

	for operatorID, record := range consensusRecordAggregates {
		var delayedResponses []string
		for duration, value := range record.delayedResponses {
			delayedResponses = append(delayedResponses, fmt.Sprintf("%s: %d\n", duration.String(), value))
		}
		c.t.AddRow(
			fmt.Sprint(operatorID),
			fmt.Sprint(record.consensusClientResponseTimeAvgTotal/time.Duration(record.consensusClientResponseTimeRecordCount)),
			strings.Join(delayedResponses, "\n"),
		)
	}

	c.t.Render()
}
