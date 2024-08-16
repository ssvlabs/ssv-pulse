package consensus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	PeerCountMeasurement = "Count"
)

type PeerMetric struct {
	metric.Base[uint32]
	url string
}

func NewPeerMetric(url, name string, healthCondition []metric.HealthCondition[uint32]) *PeerMetric {
	return &PeerMetric{
		url: url,
		Base: metric.Base[uint32]{
			HealthConditions: healthCondition,
			Name:             name,
		},
	}
}

func (p *PeerMetric) Measure() {
	var (
		resp struct {
			Data struct {
				Connected string `json:"connected"`
			} `json:"data"`
		}
	)

	res, err := http.Get(fmt.Sprintf("%s/eth/v1/node/peer_count", p.url))
	if err != nil {
		p.AddDataPoint(map[string]uint32{
			PeerCountMeasurement: 0,
		})
		logger.WriteError(metric.ConsensusGroup, p.Name, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		p.AddDataPoint(map[string]uint32{
			PeerCountMeasurement: 0,
		})

		var errorResponse any
		_ = json.NewDecoder(res.Body).Decode(&errorResponse)
		jsonErrResponse, _ := json.Marshal(errorResponse)
		logger.WriteError(
			metric.ConsensusGroup,
			p.Name,
			fmt.Errorf("received unsuccessful status code. Code: '%s'. Response: '%s'", res.Status, jsonErrResponse))
		return
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		p.AddDataPoint(map[string]uint32{
			PeerCountMeasurement: 0,
		})
		logger.WriteError(metric.ConsensusGroup, p.Name, err)
		return
	}

	peerNr, err := strconv.Atoi(resp.Data.Connected)
	if err != nil {
		p.AddDataPoint(map[string]uint32{
			PeerCountMeasurement: 0,
		})
		logger.WriteError(metric.ConsensusGroup, p.Name, err)
		return
	}

	p.AddDataPoint(map[string]uint32{
		PeerCountMeasurement: uint32(peerNr),
	})

	logger.WriteMetric(metric.ConsensusGroup, p.Name, map[string]any{"peers": peerNr})
}

func (p *PeerMetric) AggregateResults() string {
	var values []uint32
	for _, point := range p.DataPoints {
		values = append(values, point.Values[PeerCountMeasurement])
	}
	return metric.FormatPercentiles(
		metric.CalculatePercentile(values, 0),
		metric.CalculatePercentile(values, 10),
		metric.CalculatePercentile(values, 50),
		metric.CalculatePercentile(values, 90),
		metric.CalculatePercentile(values, 100))
}
