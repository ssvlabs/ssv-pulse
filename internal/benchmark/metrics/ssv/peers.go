package ssv

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ssvlabsinfra/ssv-pulse/internal/platform/logger"
	"github.com/ssvlabsinfra/ssv-pulse/internal/platform/metric"
)

const (
	PeerCountMeasurement = "Count"
)

type PeerMetric struct {
	metric.Base[uint32]
	url      string
	interval time.Duration
}

func NewPeerMetric(url, name string, interval time.Duration, healthCondition []metric.HealthCondition[uint32]) *PeerMetric {
	return &PeerMetric{
		url: url,
		Base: metric.Base[uint32]{
			HealthConditions: healthCondition,
			Name:             name,
		},
		interval: interval,
	}
}

func (p *PeerMetric) Measure(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.With("metric_name", p.Name).Debug("metric was stopped")
			return
		case <-ticker.C:
			p.measure(ctx)
		}
	}
}

func (p *PeerMetric) measure(ctx context.Context) {
	var (
		resp struct {
			Advanced struct {
				Peers uint32 `json:"peers"`
			} `json:"advanced"`
		}
	)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/node/health", p.url), nil)
	if err != nil {
		logger.WriteError(metric.SSVGroup, p.Name, err)
		return
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		p.addDataPoint(0)
		logger.WriteError(metric.SSVGroup, p.Name, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		p.addDataPoint(0)

		var errorResponse any
		_ = json.NewDecoder(res.Body).Decode(&errorResponse)
		jsonErrResponse, _ := json.Marshal(errorResponse)
		logger.WriteError(
			metric.SSVGroup,
			p.Name,
			fmt.Errorf("received unsuccessful status code. Code: '%s'. Response: '%s'", res.Status, jsonErrResponse))
		return
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		p.addDataPoint(0)
		logger.WriteError(metric.SSVGroup, p.Name, err)
		return
	}

	p.addDataPoint(resp.Advanced.Peers)
}

func (p *PeerMetric) AggregateResults() string {
	var values []uint32
	for _, point := range p.DataPoints {
		values = append(values, point.Values[PeerCountMeasurement])
	}

	percentiles := metric.CalculatePercentiles(values, 0, 10, 50, 90, 100)

	return metric.FormatPercentiles(
		percentiles[0],
		percentiles[10],
		percentiles[50],
		percentiles[90],
		percentiles[100])
}

func (p *PeerMetric) addDataPoint(value uint32) {
	p.AddDataPoint(map[string]uint32{
		PeerCountMeasurement: value,
	})
	peerCountMetric.Set(float64(value))

	logger.WriteMetric(metric.SSVGroup, p.Name, map[string]any{PeerCountMeasurement: value})
}
