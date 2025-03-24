package benchmark

import (
	"errors"
	"fmt"
	"time"

	"github.com/ssvlabs/ssv-pulse/configs"
	"github.com/ssvlabs/ssv-pulse/internal/benchmark/metrics/consensus"
	"github.com/ssvlabs/ssv-pulse/internal/benchmark/metrics/execution"
	"github.com/ssvlabs/ssv-pulse/internal/benchmark/metrics/infrastructure"
	"github.com/ssvlabs/ssv-pulse/internal/benchmark/metrics/ssv"
	"github.com/ssvlabs/ssv-pulse/internal/platform/metric"
	"github.com/ssvlabs/ssv-pulse/internal/platform/network"
)

func LoadEnabledMetrics(config configs.Config) (map[metric.Group][]metricService, error) {
	enabledMetrics := make(map[metric.Group][]metricService)

	if config.Benchmark.Consensus.Metrics.Client.Enabled {
		for i, addr := range configs.Values.Benchmark.Consensus.Addresses {
			enabledMetrics[metric.Group(metric.Group(fmt.Sprintf("%s-%d", metric.ConsensusGroup, i+1)))] = append(enabledMetrics[metric.Group(metric.Group(fmt.Sprintf("%s-%d", metric.ConsensusGroup, i+1)))],
				consensus.NewClientMetric(
					addr,
					"Client",
					[]metric.HealthCondition[string]{
						{Name: consensus.VersionMeasurement, Threshold: "", Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
					}))
		}
	}

	if config.Benchmark.Consensus.Metrics.Latency.Enabled {
		consensusClientURLs, err := config.Benchmark.Consensus.AddrURLs()
		if err != nil {
			return nil, errors.Join(err, errors.New("failed fetching Consensus client address as URL"))
		}
		for i, url := range consensusClientURLs {
			enabledMetrics[metric.Group(metric.Group(fmt.Sprintf("%s-%d", metric.ConsensusGroup, i+1)))] = append(enabledMetrics[metric.Group(metric.Group(fmt.Sprintf("%s-%d", metric.ConsensusGroup, i+1)))],
				consensus.NewLatencyMetric(
					url.Host,
					"Latency",
					time.Second*3,
					[]metric.HealthCondition[time.Duration]{
						{Name: consensus.DurationP90Measurement, Threshold: time.Second, Operator: metric.OperatorGreaterThanOrEqual, Severity: metric.SeverityHigh},
					}))
		}
	}

	if config.Benchmark.Consensus.Metrics.Peers.Enabled {
		for i, addr := range configs.Values.Benchmark.Consensus.Addresses {
			enabledMetrics[metric.Group(metric.Group(fmt.Sprintf("%s-%d", metric.ConsensusGroup, i+1)))] = append(enabledMetrics[metric.Group(metric.Group(fmt.Sprintf("%s-%d", metric.ConsensusGroup, i+1)))],
				consensus.NewPeerMetric(
					addr,
					"Peers",
					time.Second*10,
					[]metric.HealthCondition[uint32]{
						{Name: consensus.PeerCountMeasurement, Threshold: 5, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityHigh},
						{Name: consensus.PeerCountMeasurement, Threshold: 20, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityMedium},
						{Name: consensus.PeerCountMeasurement, Threshold: 40, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityLow},
					}))
		}
	}

	if config.Benchmark.Consensus.Metrics.Attestation.Enabled {
		for i, addr := range configs.Values.Benchmark.Consensus.Addresses {
			enabledMetrics[metric.Group(fmt.Sprintf("%s-%d", metric.ConsensusGroup, i+1))] = append(enabledMetrics[metric.Group(fmt.Sprintf("%s-%d", metric.ConsensusGroup, i+1))],
				consensus.NewAttestationMetric(
					addr,
					"Attestation",
					network.Supported[network.Name(config.Benchmark.Network)].GenesisTime,
					[]metric.HealthCondition[float64]{
						{Name: consensus.CorrectnessMeasurement, Threshold: 97, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityHigh},
						{Name: consensus.CorrectnessMeasurement, Threshold: 98.5, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityMedium},
					},
				))
		}
	}

	if config.Benchmark.Execution.Metrics.Peers.Enabled {
		for i, addr := range configs.Values.Benchmark.Execution.Addresses {
			enabledMetrics[metric.Group(fmt.Sprintf("%s-%d", metric.ExecutionGroup, i+1))] = append(enabledMetrics[metric.Group(fmt.Sprintf("%s-%d", metric.ExecutionGroup, i+1))],
				execution.NewPeerMetric(
					addr,
					"Peers",
					time.Second*10,
					[]metric.HealthCondition[uint32]{
						{Name: execution.PeerCountMeasurement, Threshold: 5, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityHigh},
						{Name: execution.PeerCountMeasurement, Threshold: 20, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityMedium},
						{Name: execution.PeerCountMeasurement, Threshold: 40, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityLow},
					}))
		}
	}

	if config.Benchmark.Execution.Metrics.Latency.Enabled {
		executionClientURLs, err := config.Benchmark.Execution.AddrURLs()
		if err != nil {
			return nil, errors.Join(err, errors.New("failed fetching Execution client addresses as URLs"))
		}
		for i, url := range executionClientURLs {
			enabledMetrics[metric.Group(fmt.Sprintf("%s-%d", metric.ExecutionGroup, i+1))] = append(enabledMetrics[metric.Group(fmt.Sprintf("%s-%d", metric.ExecutionGroup, i+1))],
				execution.NewLatencyMetric(
					url.Host,
					"Latency",
					time.Second*3,
					[]metric.HealthCondition[time.Duration]{
						{Name: execution.DurationP90Measurement, Threshold: time.Second, Operator: metric.OperatorGreaterThanOrEqual, Severity: metric.SeverityHigh},
					}))
		}
	}

	if config.Benchmark.SSV.Metrics.Peers.Enabled {
		enabledMetrics[metric.SSVGroup] = append(enabledMetrics[metric.SSVGroup], ssv.NewPeerMetric(
			configs.Values.Benchmark.SSV.Address,
			"Peers",
			time.Second*10,
			[]metric.HealthCondition[uint32]{
				{Name: ssv.PeerCountMeasurement, Threshold: 5, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityHigh},
				{Name: ssv.PeerCountMeasurement, Threshold: 10, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityMedium},
			}))
	}

	if config.Benchmark.SSV.Metrics.Connections.Enabled {
		enabledMetrics[metric.SSVGroup] = append(enabledMetrics[metric.SSVGroup], ssv.NewConnectionsMetric(
			configs.Values.Benchmark.SSV.Address,
			"Connections",
			time.Second*10,
			[]metric.HealthCondition[uint32]{
				{Name: ssv.InboundConnectionsMeasurement, Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
				{Name: ssv.OutboundConnectionsMeasurement, Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
			}))
	}

	if config.Benchmark.Infrastructure.Metrics.CPU.Enabled {
		enabledMetrics[metric.InfrastructureGroup] = append(enabledMetrics[metric.InfrastructureGroup],
			infrastructure.NewCPUMetric("CPU", time.Second*5, []metric.HealthCondition[float64]{}),
		)
	}

	if config.Benchmark.Infrastructure.Metrics.Memory.Enabled {
		enabledMetrics[metric.InfrastructureGroup] = append(enabledMetrics[metric.InfrastructureGroup],
			infrastructure.NewMemoryMetric("Memory", time.Second*10, []metric.HealthCondition[uint64]{
				{Name: infrastructure.FreeMemoryMeasurement, Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
			}),
		)
	}

	return enabledMetrics, nil
}
