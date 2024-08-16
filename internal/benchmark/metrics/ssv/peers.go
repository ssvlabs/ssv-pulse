package ssv

import (
	"encoding/json"
	"fmt"
	"net/http"

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
			Advanced struct {
				Peers uint32 `json:"peers"`
			} `json:"advanced"`
		}
	)
	res, err := http.Get(fmt.Sprintf("%s/v1/node/health", p.url))
	if err != nil {
		p.AddDataPoint(map[string]uint32{
			PeerCountMeasurement: 0,
		})
		logger.WriteError(metric.SSVGroup, p.Name, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		p.AddDataPoint(map[string]uint32{
			PeerCountMeasurement: 0,
		})
		logger.WriteError(metric.SSVGroup, p.Name, fmt.Errorf("received unsuccessful status code. Code: '%s'. Metric: '%s'", res.Status, p.Name))
		return
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		p.AddDataPoint(map[string]uint32{
			PeerCountMeasurement: 0,
		})
		logger.WriteError(metric.SSVGroup, p.Name, err)
		return
	}

	p.AddDataPoint(map[string]uint32{
		PeerCountMeasurement: resp.Advanced.Peers,
	})

	logger.WriteMetric(metric.SSVGroup, p.Name, map[string]any{"peers": resp.Advanced.Peers})
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
