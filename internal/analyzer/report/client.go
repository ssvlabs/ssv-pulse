package report

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aquasecurity/table"
)

var clientHeaders = []string{
	"Operator",
	"Consensus Client Response Time: avg",
	"Consensus Client Response Time: P10",
	"Consensus Client Response Time: P50",
	"Consensus Client Response Time: P90",
	"Consensus Client Response Time: delayed",
}

type ClientRecord struct {
	OperatorID uint32
	ConsensusClientResponseTimeP10,
	ConsensusClientResponseTimeP50,
	ConsensusClientResponseTimeP90,
	ConsensusClientResponseTimeAvg time.Duration
	ConsensusClientResponseTimeDelayCount map[time.Duration]uint16
}

type ClientReport struct {
	records []ClientRecord
	t       *table.Table
}

func NewClient() *ClientReport {
	t := table.New(os.Stdout)

	t.SetHeaders("Consensus Client Performance")
	t.AddHeaders(clientHeaders...)
	t.SetAutoMerge(true)
	t.SetHeaderColSpans(0, len(clientHeaders))

	var alignments []table.Alignment
	for i := 0; i < len(clientHeaders); i++ {
		alignments = append(alignments, table.AlignCenter)
	}
	t.SetAlignment(alignments...)

	return &ClientReport{
		t: t,
	}
}

func (c *ClientReport) AddRecord(record ClientRecord) {
	c.records = append(c.records, record)
}

func (c *ClientReport) Render() {
	type consensusClientRecordAggregate struct {
		delayedResponses map[time.Duration]uint16
		consensusClientResponseTimeAvgTotal,
		consensusClientResponseTimeP10Total,
		consensusClientResponseTimeP50Total,
		consensusClientResponseTimeP90Total time.Duration
		consensusClientResponseTimeRecordCount uint16
	}

	consensusClientAggregates := make(map[uint32]consensusClientRecordAggregate)

	for _, record := range c.records {
		aggregate := consensusClientAggregates[record.OperatorID]

		for duration, value := range record.ConsensusClientResponseTimeDelayCount {
			_, ok := aggregate.delayedResponses[duration]
			if !ok {
				aggregate.delayedResponses = make(map[time.Duration]uint16)
			}
			aggregate.delayedResponses[duration] += value
		}

		aggregate.consensusClientResponseTimeAvgTotal += record.ConsensusClientResponseTimeAvg
		aggregate.consensusClientResponseTimeP10Total += record.ConsensusClientResponseTimeP10
		aggregate.consensusClientResponseTimeP50Total += record.ConsensusClientResponseTimeP50
		aggregate.consensusClientResponseTimeP90Total += record.ConsensusClientResponseTimeP90
		aggregate.consensusClientResponseTimeRecordCount++

		consensusClientAggregates[record.OperatorID] = aggregate
	}

	for operatorID, record := range consensusClientAggregates {
		var delayedResponses []string
		for duration, value := range record.delayedResponses {
			delayedResponses = append(delayedResponses, fmt.Sprintf("%s: %d\n", duration.String(), value))
		}
		c.t.AddRow(
			fmt.Sprint(operatorID),
			fmt.Sprint(record.consensusClientResponseTimeAvgTotal/time.Duration(record.consensusClientResponseTimeRecordCount)),
			fmt.Sprint(record.consensusClientResponseTimeP10Total/time.Duration(record.consensusClientResponseTimeRecordCount)),
			fmt.Sprint(record.consensusClientResponseTimeP50Total/time.Duration(record.consensusClientResponseTimeRecordCount)),
			fmt.Sprint(record.consensusClientResponseTimeP90Total/time.Duration(record.consensusClientResponseTimeRecordCount)),
			strings.Join(delayedResponses, "\n"),
		)
	}

	c.t.Render()
}
