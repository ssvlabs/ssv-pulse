package execution

import (
	"context"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	Peers metric.Name = "Peers"
)

type Service struct {
	interval   time.Duration
	peerMetric *PeerMetric
}

func New(apiURL string) *Service {
	return &Service{
		peerMetric: NewPeerMetric(apiURL),
		interval:   time.Second * 5,
	}
}

func (s *Service) Start(ctx context.Context) (map[metric.Name]metric.Result, error) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			peers, err := s.peerMetric.Get()
			if err != nil {
				logger.WriteError(metric.ExecutionGroup, Peers, err)
			} else {
				logger.WriteMetric(metric.ExecutionGroup, Peers, map[string]any{"peers": peers})
			}
		case <-ctx.Done():
			return map[metric.Name]metric.Result{
				Peers: {
					Value:    []byte(metric.FormatPercentiles(s.peerMetric.Aggregate())),
					Health:   "",
					Severity: "",
				},
			}, ctx.Err()
		}
	}
}
