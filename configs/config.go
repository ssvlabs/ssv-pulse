package configs

import (
	"errors"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/network"
)

var Values Config

type MetricConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type ConsensusMetrics struct {
	Client  MetricConfig `mapstructure:"client"`
	Latency MetricConfig `mapstructure:"latency"`
	Peers   MetricConfig `mapstructure:"peers"`
}

type ExecutionMetrics struct {
	Peers MetricConfig `mapstructure:"peers"`
}

type SsvMetrics struct {
	Peers MetricConfig `mapstructure:"peers"`
}

type InfrastructureMetrics struct {
	CPU    MetricConfig `mapstructure:"cpu"`
	Memory MetricConfig `mapstructure:"memory"`
}

type ConsensusConfig struct {
	Address string           `mapstructure:"address"`
	Metrics ConsensusMetrics `mapstructure:"metrics"`
}

type ExecutionConfig struct {
	Address string           `mapstructure:"address"`
	Metrics ExecutionMetrics `mapstructure:"metrics"`
}

type SsvConfig struct {
	Address string     `mapstructure:"address"`
	Metrics SsvMetrics `mapstructure:"metrics"`
}

type InfrastructureConfig struct {
	Metrics InfrastructureMetrics `mapstructure:"metrics"`
}

type BenchmarkConfig struct {
	Consensus      ConsensusConfig      `mapstructure:"consensus"`
	Execution      ExecutionConfig      `mapstructure:"execution"`
	Ssv            SsvConfig            `mapstructure:"ssv"`
	Infrastructure InfrastructureConfig `mapstructure:"infrastructure"`
	Duration       time.Duration        `mapstructure:"duration"`
	Network        string               `mapstructure:"network"`
}

func (bc BenchmarkConfig) Validate() (bool, error) {
	if bc.Consensus.Address == "" {
		return false, errors.New("consensus client address was not valid")
	}

	if bc.Execution.Address == "" {
		return false, errors.New("execution client address was not valid")
	}

	if bc.Ssv.Address == "" {
		return false, errors.New("SSV client address was not valid")
	}

	network := network.Name(bc.Network)
	if err := network.Validate(); err != nil {
		return false, errors.Join(err, errors.New("network name was not valid"))
	}

	return true, nil
}

type Config struct {
	Benchmark BenchmarkConfig `mapstructure:"benchmark"`
}
