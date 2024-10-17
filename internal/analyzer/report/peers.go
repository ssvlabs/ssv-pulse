package report

import (
	"fmt"
	"os"

	"github.com/aquasecurity/table"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
	"github.com/ssvlabs/ssv-pulse/internal/platform/array"
)

var peersHeaders = []string{
	"Operator",
	"Peers: ID",
	"Peers: avg",
	"Peers: \n ssv client versions",
}

type PeerRecord struct {
	OperatorID             uint32
	PeerID                 string
	PeerCountAvg           parser.Metric[float64]
	PeersSSVClientVersions []string
}

type PeersReport struct {
	records []PeerRecord
	t       *table.Table
}

func NewPeers() *PeersReport {
	t := table.New(os.Stdout)

	t.SetHeaders("Peers Performance")
	t.AddHeaders(peersHeaders...)
	t.SetAutoMerge(true)
	t.SetHeaderColSpans(0, len(peersHeaders))

	var alignments []table.Alignment
	for i := 0; i < len(peersHeaders); i++ {
		alignments = append(alignments, table.AlignCenter)
	}
	t.SetAlignment(alignments...)

	return &PeersReport{
		t: t,
	}
}

func (p *PeersReport) AddRecord(record PeerRecord) {
	p.records = append(p.records, record)
}

func (p *PeersReport) Render() {
	type peerRecordAggregate struct {
		peerID                string
		peerCountFound        bool
		peerCountAvgTotal     float64
		peerCountRecordCount  uint16
		peersSsvClientVersion []string
	}

	peerRecordsAggregates := make(map[uint32]peerRecordAggregate)

	for _, record := range p.records {
		aggregate := peerRecordsAggregates[record.OperatorID]
		aggregate.peerCountAvgTotal += record.PeerCountAvg.Value
		if !peerRecordsAggregates[record.OperatorID].peerCountFound {
			aggregate.peerCountFound = record.PeerCountAvg.Found
		}
		aggregate.peerCountRecordCount++
		aggregate.peerID = record.PeerID
		aggregate.peersSsvClientVersion = array.CollectDistinct(peerRecordsAggregates[record.OperatorID].peersSsvClientVersion, record.PeersSSVClientVersions)

		peerRecordsAggregates[record.OperatorID] = aggregate
	}

	for operatorID, record := range peerRecordsAggregates {
		var peerCountRecord string
		if record.peerCountRecordCount != 0 {
			peerCountRecord = fmt.Sprint(record.peerCountAvgTotal / float64(record.peerCountRecordCount))
		}
		if !record.peerCountFound {
			peerCountRecord = "n/a"
		}

		p.t.AddRow(
			fmt.Sprint(operatorID),
			record.peerID,
			peerCountRecord,
			fmt.Sprint(record.peersSsvClientVersion),
		)
	}
	p.t.Render()
}
