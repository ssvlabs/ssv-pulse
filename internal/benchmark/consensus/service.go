package consensus

import (
	"context"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	Peers   metric.Name = "Peers"
	Latency metric.Name = "Latency"
	Client  metric.Name = "Client"
)

type (
	Service struct {
		interval      time.Duration
		peerMetric    *PeerMetric
		latencyMetric *LatencyMetric
		clientMetric  *ClientVersionMetric
	}
)

func New(apiURL string) *Service {
	return &Service{
		interval:      time.Second * 5,
		peerMetric:    NewPeerMetric(apiURL),
		latencyMetric: NewLatencyMetric(apiURL),
		clientMetric:  NewClientVersionMetric(apiURL),
	}
}

func (s *Service) Start(ctx context.Context) (map[metric.Name]metric.Result, error) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	clientVersion, err := s.clientMetric.Get()
	if err != nil {
		logger.WriteError(metric.ConsensusGroup, Client, err)
	} else {
		logger.WriteMetric(metric.ConsensusGroup, Client, map[string]any{
			"client_version": clientVersion,
		})
	}

	for {
		select {
		case <-ticker.C:
			peers, err := s.peerMetric.Get()
			if err != nil {
				logger.WriteError(metric.ConsensusGroup, Peers, err)
			} else {
				logger.WriteMetric(metric.ConsensusGroup, Peers, map[string]any{"peers": peers})
			}

			latency, err := s.latencyMetric.Get()
			if err != nil {
				logger.WriteError(metric.ConsensusGroup, Latency, err)
			} else {
				logger.WriteMetric(metric.ConsensusGroup, Latency, map[string]any{
					"latency_ms": latency.Milliseconds(),
				})
			}
		case <-ctx.Done():
			return map[metric.Name]metric.Result{
				Peers: {
					Value:    []byte(metric.FormatPercentiles(s.peerMetric.Aggregate())),
					Health:   "",
					Severity: "",
				},
				Latency: {
					Value:    []byte(metric.FormatPercentiles(s.latencyMetric.Aggregate())),
					Health:   "",
					Severity: "",
				},
				Client: {
					Value: []byte(clientVersion),
				},
			}, ctx.Err()
		}
	}
}
