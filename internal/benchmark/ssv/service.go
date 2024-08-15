package ssv

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
	peerMetric *PeerMetric
	interval   time.Duration
}

func New(url string) *Service {
	return &Service{
		peerMetric: NewPeerMetric(url, []metric.HealthCondition[uint32]{
			{Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
			{Threshold: 50, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityMedium},
		}),
		interval: time.Second * 5,
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
				logger.WriteError(metric.SSVGroup, Peers, err)
			} else {
				logger.WriteMetric(metric.SSVGroup, Peers, map[string]any{"peers": peers})
			}
		case <-ctx.Done():
			health, severity := s.peerMetric.Health()
			return map[metric.Name]metric.Result{
				Peers: {
					Value:    []byte(metric.FormatPercentiles(s.peerMetric.Aggregate())),
					Health:   health,
					Severity: severity,
				},
			}, ctx.Err()
		}
	}
}
