package benchmark

import (
	"time"

	"github.com/ssvlabs/ssv-benchmark/configs"
	"github.com/ssvlabs/ssv-benchmark/internal/benchmark/metrics/consensus"
	"github.com/ssvlabs/ssv-benchmark/internal/benchmark/metrics/execution"
	"github.com/ssvlabs/ssv-benchmark/internal/benchmark/metrics/infrastructure"
	"github.com/ssvlabs/ssv-benchmark/internal/benchmark/metrics/ssv"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/network"
)

func LoadEnabledMetrics(config configs.Config) map[metric.Group][]metricService {
	enabledMetrics := make(map[metric.Group][]metricService)

	if config.Benchmark.Consensus.Metrics.Client.Enabled {
		enabledMetrics[metric.ConsensusGroup] = append(enabledMetrics[metric.ConsensusGroup], consensus.NewClientMetric(
			configs.Values.Benchmark.Consensus.Address,
			"Client",
			[]metric.HealthCondition[string]{
				{Name: consensus.Version, Threshold: "", Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
			}))
	}

	if config.Benchmark.Consensus.Metrics.Latency.Enabled {
		enabledMetrics[metric.ConsensusGroup] = append(enabledMetrics[metric.ConsensusGroup],
			consensus.NewLatencyMetric(configs.Values.Benchmark.Consensus.Address, "Latency", []metric.HealthCondition[time.Duration]{}),
		)
	}

	if config.Benchmark.Consensus.Metrics.Peers.Enabled {
		enabledMetrics[metric.ConsensusGroup] = append(enabledMetrics[metric.ConsensusGroup], consensus.NewPeerMetric(
			configs.Values.Benchmark.Consensus.Address,
			"Peers",
			[]metric.HealthCondition[uint32]{
				{Name: consensus.PeerCountMeasurement, Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
				{Name: consensus.PeerCountMeasurement, Threshold: 50, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityMedium},
			}))
	}

	if config.Benchmark.Consensus.Metrics.Attestation.Enabled {
		enabledMetrics[metric.ConsensusGroup] = append(enabledMetrics[metric.ConsensusGroup], consensus.NewAttestationMetric(
			configs.Values.Benchmark.Consensus.Address,
			"Attestation",
			network.GenesisTime[network.Name(config.Benchmark.Network)],
			[]metric.HealthCondition[float64]{},
		))
	}

	if config.Benchmark.Execution.Metrics.Peers.Enabled {
		enabledMetrics[metric.ExecutionGroup] = append(enabledMetrics[metric.ExecutionGroup], execution.NewPeerMetric(
			configs.Values.Benchmark.Execution.Address,
			"Peers",
			[]metric.HealthCondition[uint32]{
				{Name: execution.PeerCountMeasurement, Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
				{Name: execution.PeerCountMeasurement, Threshold: 50, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityMedium},
			}))
	}

	if config.Benchmark.Ssv.Metrics.Peers.Enabled {
		enabledMetrics[metric.SSVGroup] = append(enabledMetrics[metric.SSVGroup], ssv.NewPeerMetric(
			configs.Values.Benchmark.Ssv.Address,
			"Peers",
			[]metric.HealthCondition[uint32]{
				{Name: ssv.PeerCountMeasurement, Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
				{Name: ssv.PeerCountMeasurement, Threshold: 50, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityMedium},
			}))
	}

	if config.Benchmark.Infrastructure.Metrics.CPU.Enabled {
		enabledMetrics[metric.InfrastructureGroup] = append(enabledMetrics[metric.InfrastructureGroup],
			infrastructure.NewCPUMetric("CPU", []metric.HealthCondition[float64]{}),
		)
	}

	if config.Benchmark.Infrastructure.Metrics.Memory.Enabled {
		enabledMetrics[metric.InfrastructureGroup] = append(enabledMetrics[metric.InfrastructureGroup],
			infrastructure.NewMemoryMetric("Memory", []metric.HealthCondition[uint64]{
				{Name: infrastructure.FreeMemoryMeasurement, Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
			}),
		)
	}
	return enabledMetrics
}
