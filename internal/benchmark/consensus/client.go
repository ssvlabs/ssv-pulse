package consensus

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	Version = "Version"
)

type ClientMetric struct {
	metric.Base[string]
	url        string
	isMeasured bool
}

func NewClientMetric(url, name string, healthCondition []metric.HealthCondition[string]) *ClientMetric {
	return &ClientMetric{
		url: url,
		Base: metric.Base[string]{
			HealthConditions: healthCondition,
			Name:             name,
		},
	}
}

func (c *ClientMetric) Measure() {
	var (
		resp struct {
			Data struct {
				Version string `json:"version"`
			} `json:"data"`
		}
	)
	if c.isMeasured {
		return
	}
	res, err := http.Get(fmt.Sprintf("%s/eth/v1/node/version", c.url))
	if err != nil {
		c.AddDataPoint(map[string]string{
			Version: "",
		})
		logger.WriteError(metric.ConsensusGroup, c.Name, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		c.AddDataPoint(map[string]string{
			Version: "",
		})
		logger.WriteError(metric.ConsensusGroup, c.Name, fmt.Errorf("received unsuccessful status code. Code: '%s'. Metric: '%s'", res.Status, c.Name))
		return
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		c.AddDataPoint(map[string]string{
			Version: "",
		})
		logger.WriteError(metric.ConsensusGroup, c.Name, err)
		return
	}

	c.AddDataPoint(map[string]string{
		Version: resp.Data.Version,
	})
	c.isMeasured = true
	logger.WriteMetric(metric.ConsensusGroup, c.Name, map[string]any{"version": resp.Data.Version})
}

func (p *ClientMetric) AggregateResults() string {
	return p.DataPoints[0].Values[Version]
}
