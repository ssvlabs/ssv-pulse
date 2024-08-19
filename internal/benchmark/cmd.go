package benchmark

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabs/ssv-benchmark/configs"
	"github.com/ssvlabs/ssv-benchmark/internal/benchmark/metrics/consensus"
	"github.com/ssvlabs/ssv-benchmark/internal/benchmark/metrics/execution"
	"github.com/ssvlabs/ssv-benchmark/internal/benchmark/metrics/infrastructure"
	"github.com/ssvlabs/ssv-benchmark/internal/benchmark/metrics/ssv"
	"github.com/ssvlabs/ssv-benchmark/internal/benchmark/report"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/cmd"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/lifecycle"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	durationFlag = "duration"

	consensusAddrFlag          = "consensus-addr"
	consensusMetricClientFlag  = "consensus-metric-client-enabled"
	consensusMetricLatencyFlag = "consensus-metric-latency-enabled"
	consensusMetricPeersFlag   = "consensus-metric-peers-enabled"

	executionAddrFlag        = "execution-addr"
	executionMetricPeersFlag = "execution-metric-peers-enabled"

	ssvAddrFlag        = "ssv-addr"
	ssvMetricPeersFlag = "ssv-metric-peers-enabled"

	infraMetricCPUFlag    = "infra-metric-cpu-enabled"
	infraMetricMemoryFlag = "infra-metric-memory-enabled"

	networkFlag              = "network"
	defaultExecutionDuration = time.Second * 60 * 5
)

func init() {

}

var CMD = &cobra.Command{
	Use:   "benchmark",
	Short: "Run benchmarks of ssv node",
	RunE: func(cobraCMD *cobra.Command, args []string) error {
		addFlags(cobraCMD)
		if err := bindFlags(cobraCMD); err != nil {
			panic(err.Error())
		}
		slog.
			With("config_file", viper.ConfigFileUsed()).
			With("config", configs.Values).
			Debug("configurations loaded")

		ctx, cancel := context.WithTimeout(context.Background(), configs.Values.Benchmark.Duration)

		isValid, err := configs.Values.Benchmark.Validate()
		if !isValid {
			panic(err.Error())
		}

		benchmarkService := New(map[metric.Group][]metricService{
			metric.ConsensusGroup: {
				consensus.NewLatencyMetric(configs.Values.Benchmark.Consensus.Address, "Latency", []metric.HealthCondition[time.Duration]{}),
				consensus.NewPeerMetric(configs.Values.Benchmark.Consensus.Address, "Peers", []metric.HealthCondition[uint32]{
					{Name: consensus.PeerCountMeasurement, Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
					{Name: consensus.PeerCountMeasurement, Threshold: 50, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityMedium},
				}),
				consensus.NewClientMetric(configs.Values.Benchmark.Consensus.Address, "Client", []metric.HealthCondition[string]{
					{Name: consensus.Version, Threshold: "", Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
				}),
			},

			metric.ExecutionGroup: {
				execution.NewPeerMetric(configs.Values.Benchmark.Execution.Address, "Peers", []metric.HealthCondition[uint32]{
					{Name: execution.PeerCountMeasurement, Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
					{Name: execution.PeerCountMeasurement, Threshold: 50, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityMedium},
				}),
			},

			metric.SSVGroup: {
				ssv.NewPeerMetric(configs.Values.Benchmark.Ssv.Address, "Peers", []metric.HealthCondition[uint32]{
					{Name: ssv.PeerCountMeasurement, Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
					{Name: ssv.PeerCountMeasurement, Threshold: 50, Operator: metric.OperatorLessThanOrEqual, Severity: metric.SeverityMedium},
				}),
			},
			metric.InfrastructureGroup: {
				infrastructure.NewMemoryMetric("Memory", []metric.HealthCondition[uint64]{
					{Name: infrastructure.FreeMemoryMeasurement, Threshold: 0, Operator: metric.OperatorEqual, Severity: metric.SeverityHigh},
				}),
				infrastructure.NewCPUMetric("CPU", []metric.HealthCondition[float64]{}),
			},
		}, report.New())

		go benchmarkService.Start(ctx)

		lifecycle.ListenForApplicationShutDown(ctx, func() {
			cancel()
			slog.Warn("terminating the application")
		}, make(chan os.Signal))
		return nil
	},
}

func addFlags(cobraCMD *cobra.Command) {
	cmd.AddPersistentDurationFlag(cobraCMD, durationFlag, defaultExecutionDuration, "Duration for which the application will run to gather metrics, e.g. '5m'", false)
	cmd.AddPersistentStringFlag(cobraCMD, consensusAddrFlag, "", "Consensus client address (beacon node API) with scheme (HTTP/HTTPS) and port, e.g. https://lighthouse:5052", true)
	cmd.AddPersistentBoolFlag(cobraCMD, consensusMetricClientFlag, true, "Enable consensus client metric", false)
	cmd.AddPersistentBoolFlag(cobraCMD, consensusMetricLatencyFlag, true, "Enable consensus latency metric", false)
	cmd.AddPersistentBoolFlag(cobraCMD, consensusMetricPeersFlag, true, "Enable consensus peers metric", false)

	cmd.AddPersistentStringFlag(cobraCMD, executionAddrFlag, "", "Execution client address with scheme (HTTP/HTTPS) and port, e.g. https://geth:8545", true)
	cmd.AddPersistentBoolFlag(cobraCMD, executionMetricPeersFlag, true, "Enable execution peers metric", false)

	cmd.AddPersistentStringFlag(cobraCMD, ssvAddrFlag, "", "SSV API address with scheme (HTTP/HTTPS) and port, e.g. http://ssv-node:16000", true)
	cmd.AddPersistentBoolFlag(cobraCMD, ssvMetricPeersFlag, true, "Enable SSV peers metric", false)

	cmd.AddPersistentBoolFlag(cobraCMD, infraMetricCPUFlag, true, "Enable infrastructure CPU metric", false)
	cmd.AddPersistentBoolFlag(cobraCMD, infraMetricMemoryFlag, true, "Enable infrastructure memory metric", false)

	cmd.AddPersistentStringFlag(cobraCMD, networkFlag, "", "Ethereum network to use, either 'mainnet' or 'holesky'", true)
}

func bindFlags(cmd *cobra.Command) error {
	if err := viper.BindPFlag("benchmark.execution-duration", cmd.PersistentFlags().Lookup(durationFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("benchmark.consensus.address", cmd.PersistentFlags().Lookup(consensusAddrFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("benchmark.execution.address", cmd.PersistentFlags().Lookup(executionAddrFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("benchmark.ssv.address", cmd.PersistentFlags().Lookup(ssvAddrFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("benchmark.network", cmd.PersistentFlags().Lookup(networkFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("benchmark.consensus.metrics.client.enabled", cmd.PersistentFlags().Lookup(consensusMetricClientFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("benchmark.consensus.metrics.latency.enabled", cmd.PersistentFlags().Lookup(consensusMetricLatencyFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("benchmark.consensus.metrics.peers.enabled", cmd.PersistentFlags().Lookup(consensusMetricPeersFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("benchmark.execution.metrics.peers.enabled", cmd.PersistentFlags().Lookup(executionMetricPeersFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("benchmark.ssv.metrics.peers.enabled", cmd.PersistentFlags().Lookup(ssvMetricPeersFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("benchmark.infrastructure.metrics.cpu.enabled", cmd.PersistentFlags().Lookup(infraMetricCPUFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("benchmark.infrastructure.metrics.memory.enabled", cmd.PersistentFlags().Lookup(infraMetricMemoryFlag)); err != nil {
		return err
	}
	return nil
}
