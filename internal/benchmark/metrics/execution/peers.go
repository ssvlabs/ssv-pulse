package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
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
			Result string `json:"result"`
		}
	)

	request := struct {
		Jsonrpc string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  []any  `json:"params"`
		ID      int    `json:"id"`
	}{
		Jsonrpc: "2.0",
		Method:  "net_peerCount",
		Params:  []any{},
		ID:      1,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		logger.WriteError(metric.ExecutionGroup, p.Name, err)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewBuffer(requestBytes))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		logger.WriteError(metric.ExecutionGroup, p.Name, err)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		p.AddDataPoint(map[string]uint32{
			PeerCountMeasurement: 0,
		})
		logger.WriteError(metric.ExecutionGroup, p.Name, err)
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
			metric.ExecutionGroup,
			p.Name,
			fmt.Errorf("received unsuccessful status code. Code: '%s'. Response: '%s'", res.Status, jsonErrResponse))
		return
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		p.AddDataPoint(map[string]uint32{
			PeerCountMeasurement: 0,
		})
		logger.WriteError(metric.ExecutionGroup, p.Name, err)
		return
	}

	peerCountHex := resp.Result
	if peerCountHex == "" {
		p.AddDataPoint(map[string]uint32{
			PeerCountMeasurement: 0,
		})
		logger.WriteError(metric.ExecutionGroup, p.Name, errors.New("peer count RPC response was empty. Most likely net_peerCount RPC method is not supported"))
		return
	}

	peerCount, err := strconv.ParseInt(peerCountHex[2:], 16, 64)
	if err != nil {
		p.AddDataPoint(map[string]uint32{
			PeerCountMeasurement: 0,
		})
		logger.WriteError(metric.ExecutionGroup, p.Name, err)
		return
	}

	p.AddDataPoint(map[string]uint32{
		PeerCountMeasurement: uint32(peerCount),
	})

	logger.WriteMetric(metric.ExecutionGroup, p.Name, map[string]any{PeerCountMeasurement: peerCount})
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
