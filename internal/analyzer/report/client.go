package report

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/aquasecurity/table"
)

var (
	consensusClientHeaders = []string{
		"Operator",
		"Response Time: \n avg",
		"Response Time: \n P10",
		"Response Time: \n P50",
		"Response Time: \n P90",
		"Response Time: \n delayed",
	}

	ssvClientHeaders = []string{
		"Crashes(reason: unhealthy EL/CL): \n total(EL/CL)",
	}
)

type ClientRecord struct {
	OperatorID uint32
	ConsensusClientResponseTimeP10,
	ConsensusClientResponseTimeP50,
	ConsensusClientResponseTimeP90,
	ConsensusClientResponseTimeAvg time.Duration
	ConsensusClientResponseTimeDelayPercent map[time.Duration]float32

	SSVClientCrashesTotal,
	SSVClientELUnhealthy,
	SSVClientCLUnhealthy uint16
}

type ClientReport struct {
	records []ClientRecord
	t       *table.Table
}

func NewClient() *ClientReport {
	t := table.New(os.Stdout)

	headers := slices.Concat(consensusClientHeaders, ssvClientHeaders)
	t.SetHeaders("Consensus Client Performance", "SSV Client Performance")
	t.AddHeaders(headers...)
	t.SetAutoMerge(true)
	t.SetHeaderColSpans(0, len(consensusClientHeaders), len(ssvClientHeaders))

	var alignments []table.Alignment
	for i := 0; i < len(headers); i++ {
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
		delayedResponsesPercentTotal map[time.Duration]float32
		delayedResponsesPercentCount map[time.Duration]uint32
		consensusClientResponseTimeAvgTotal,
		consensusClientResponseTimeP10Total,
		consensusClientResponseTimeP50Total,
		consensusClientResponseTimeP90Total time.Duration
		consensusClientResponseTimeRecordCount,

		ssvClientCrashesTotal,
		SSVClientELUnhealthyTotal,
		SSVClientCLUnhealthyTotal uint16
	}

	consensusClientAggregates := make(map[uint32]consensusClientRecordAggregate)

	for _, record := range c.records {
		aggregate := consensusClientAggregates[record.OperatorID]

		for duration, value := range record.ConsensusClientResponseTimeDelayPercent {
			_, ok := aggregate.delayedResponsesPercentTotal[duration]
			if !ok {
				aggregate.delayedResponsesPercentTotal = make(map[time.Duration]float32)
			}
			_, ok = aggregate.delayedResponsesPercentCount[duration]
			if !ok {
				aggregate.delayedResponsesPercentCount = make(map[time.Duration]uint32)
			}
			aggregate.delayedResponsesPercentTotal[duration] += value
			aggregate.delayedResponsesPercentCount[duration]++
		}

		aggregate.consensusClientResponseTimeAvgTotal += record.ConsensusClientResponseTimeAvg
		aggregate.consensusClientResponseTimeP10Total += record.ConsensusClientResponseTimeP10
		aggregate.consensusClientResponseTimeP50Total += record.ConsensusClientResponseTimeP50
		aggregate.consensusClientResponseTimeP90Total += record.ConsensusClientResponseTimeP90
		aggregate.consensusClientResponseTimeRecordCount++
		aggregate.ssvClientCrashesTotal += record.SSVClientCrashesTotal
		aggregate.SSVClientCLUnhealthyTotal += record.SSVClientCLUnhealthy
		aggregate.SSVClientELUnhealthyTotal += record.SSVClientELUnhealthy

		consensusClientAggregates[record.OperatorID] = aggregate
	}

	for operatorID, record := range consensusClientAggregates {
		var delayedResponses []string
		for duration, value := range record.delayedResponsesPercentTotal {
			avgPercent := value / float32(record.delayedResponsesPercentCount[duration])
			delayedResponses = append(delayedResponses, fmt.Sprintf("%s: %.2f%% \n", duration.String(), avgPercent))
		}
		c.t.AddRow(
			fmt.Sprint(operatorID),
			fmt.Sprint(record.consensusClientResponseTimeAvgTotal/time.Duration(record.consensusClientResponseTimeRecordCount)),
			fmt.Sprint(record.consensusClientResponseTimeP10Total/time.Duration(record.consensusClientResponseTimeRecordCount)),
			fmt.Sprint(record.consensusClientResponseTimeP50Total/time.Duration(record.consensusClientResponseTimeRecordCount)),
			fmt.Sprint(record.consensusClientResponseTimeP90Total/time.Duration(record.consensusClientResponseTimeRecordCount)),
			strings.Join(delayedResponses, "\n"),
			fmt.Sprintf("%d(%d/%d)", record.ssvClientCrashesTotal, record.SSVClientELUnhealthyTotal, record.SSVClientCLUnhealthyTotal),
		)
	}

	c.t.Render()
}
