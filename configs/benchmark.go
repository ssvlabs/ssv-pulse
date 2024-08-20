package configs

import (
	"errors"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/network"
)

type Metric struct {
	Enabled bool `mapstructure:"enabled"`
}

type ConsensusMetrics struct {
	Client  Metric `mapstructure:"client"`
	Latency Metric `mapstructure:"latency"`
	Peers   Metric `mapstructure:"peers"`
}

type ExecutionMetrics struct {
	Peers Metric `mapstructure:"peers"`
}

type SsvMetrics struct {
	Peers Metric `mapstructure:"peers"`
}

type InfrastructureMetrics struct {
	CPU    Metric `mapstructure:"cpu"`
	Memory Metric `mapstructure:"memory"`
}

type Consensus struct {
	Address string           `mapstructure:"address"`
	Metrics ConsensusMetrics `mapstructure:"metrics"`
}

type Execution struct {
	Address string           `mapstructure:"address"`
	Metrics ExecutionMetrics `mapstructure:"metrics"`
}

type SSV struct {
	Address string     `mapstructure:"address"`
	Metrics SsvMetrics `mapstructure:"metrics"`
}

type Infrastructure struct {
	Metrics InfrastructureMetrics `mapstructure:"metrics"`
}

type Benchmark struct {
	Consensus      Consensus      `mapstructure:"consensus"`
	Execution      Execution      `mapstructure:"execution"`
	Ssv            SSV            `mapstructure:"ssv"`
	Infrastructure Infrastructure `mapstructure:"infrastructure"`
	Duration       time.Duration  `mapstructure:"duration"`
	Network        string         `mapstructure:"network"`
}

func (b Benchmark) Validate() (bool, error) {
	if b.Consensus.Address == "" {
		return false, errors.New("consensus client address was empty")
	}

	if b.Execution.Address == "" {
		return false, errors.New("execution client address was empty")
	}

	if b.Ssv.Address == "" {
		return false, errors.New("SSV client address was empty")
	}

	network := network.Name(b.Network)
	if err := network.Validate(); err != nil {
		return false, errors.Join(err, errors.New("network name was not valid"))
	}

	return true, nil
}
