package ssv

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/ssvlabsinfra/ssv-pulse/internal/platform/logger"
	"github.com/ssvlabsinfra/ssv-pulse/internal/platform/metric"
)

const (
	InboundConnectionsMeasurement  = "InboundConnections"
	OutboundConnectionsMeasurement = "OutboundConnections"
)

type ConnectionsMetric struct {
	metric.Base[uint32]
	url      string
	interval time.Duration
}

func NewConnectionsMetric(url, name string, interval time.Duration, healthCondition []metric.HealthCondition[uint32]) *ConnectionsMetric {
	return &ConnectionsMetric{
		url: url,
		Base: metric.Base[uint32]{
			HealthConditions: healthCondition,
			Name:             name,
		},
		interval: interval,
	}
}

func (p *ConnectionsMetric) Measure(ctx context.Context) {
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

func (c *ConnectionsMetric) measure(ctx context.Context) {
	var (
		resp struct {
			Advanced struct {
				Inbound  uint32 `json:"inbound_conns"`
				Outbound uint32 `json:"outbound_conns"`
			} `json:"advanced"`
		}
	)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/node/health", c.url), nil)
	if err != nil {
		logger.WriteError(metric.SSVGroup, c.Name, err)
		return
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if err != ctx.Err() {
			c.writeMetric(0, 0)

			logger.WriteError(metric.SSVGroup, c.Name, err)
		}
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		c.writeMetric(0, 0)

		var errorResponse any
		_ = json.NewDecoder(res.Body).Decode(&errorResponse)
		jsonErrResponse, _ := json.Marshal(errorResponse)
		logger.WriteError(
			metric.SSVGroup,
			c.Name,
			fmt.Errorf("received unsuccessful status code. Code: '%s'. Response: '%s'", res.Status, jsonErrResponse))
		return
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		c.writeMetric(0, 0)

		logger.WriteError(metric.SSVGroup, c.Name, err)
		return
	}

	c.writeMetric(resp.Advanced.Inbound, resp.Advanced.Outbound)
}

func (c *ConnectionsMetric) writeMetric(inbound, outbound uint32) {
	c.AddDataPoint(map[string]uint32{
		InboundConnectionsMeasurement:  inbound,
		OutboundConnectionsMeasurement: outbound,
	})

	connectionsMetric.With(prometheus.Labels{connectionDirectionLabel: "inbound"}).Set(float64(inbound))
	connectionsMetric.With(prometheus.Labels{connectionDirectionLabel: "outbound"}).Set(float64(outbound))

	logger.WriteMetric(metric.SSVGroup, c.Name, map[string]any{
		InboundConnectionsMeasurement:  inbound,
		OutboundConnectionsMeasurement: outbound})
}

func (p *ConnectionsMetric) AggregateResults() string {
	var measurements map[string][]uint32 = make(map[string][]uint32)

	for _, point := range p.DataPoints {
		measurements[InboundConnectionsMeasurement] = append(measurements[InboundConnectionsMeasurement], point.Values[InboundConnectionsMeasurement])
		measurements[OutboundConnectionsMeasurement] = append(measurements[OutboundConnectionsMeasurement], point.Values[OutboundConnectionsMeasurement])
	}

	inboundPercentiles := metric.CalculatePercentiles(measurements[InboundConnectionsMeasurement], 0, 50)
	outboundPercentiles := metric.CalculatePercentiles(measurements[OutboundConnectionsMeasurement], 0, 50)

	return fmt.Sprintf("inbound_min=%d, inbound_P50=%d, outbound_min=%d, outbound_P50=%d",
		inboundPercentiles[0],
		inboundPercentiles[50],
		outboundPercentiles[0],
		outboundPercentiles[50],
	)
}
