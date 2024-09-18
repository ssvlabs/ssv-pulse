package configs

import (
	"errors"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/platform/network"
)

type Metric struct {
	Enabled bool `mapstructure:"enabled"`
}

type ConsensusMetrics struct {
	Client      Metric `mapstructure:"client"`
	Latency     Metric `mapstructure:"latency"`
	Peers       Metric `mapstructure:"peers"`
	Attestation Metric `mapstructure:"attestation"`
}

type ExecutionMetrics struct {
	Peers Metric `mapstructure:"peers"`
}

type SSVMetrics struct {
	Peers       Metric `mapstructure:"peers"`
	Connections Metric `mapstructure:"connections"`
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
	Metrics SSVMetrics `mapstructure:"metrics"`
}

type Infrastructure struct {
	Metrics InfrastructureMetrics `mapstructure:"metrics"`
}

type Server struct {
	Port uint16 `mapstructure:"port"`
}

type Benchmark struct {
	Consensus      Consensus      `mapstructure:"consensus"`
	Execution      Execution      `mapstructure:"execution"`
	SSV            SSV            `mapstructure:"ssv"`
	Infrastructure Infrastructure `mapstructure:"infrastructure"`
	Server         Server         `mapstructure:"server"`
	Duration       time.Duration  `mapstructure:"duration"`
	Network        string         `mapstructure:"network"`
}

func (b Benchmark) Validate() (bool, error) {
	if b.Consensus.Metrics.Peers.Enabled ||
		b.Consensus.Metrics.Attestation.Enabled ||
		b.Consensus.Metrics.Client.Enabled ||
		b.Consensus.Metrics.Latency.Enabled {
		if err := validateURL(b.Consensus.Address); err != nil {
			return false, errors.Join(err, errors.New("consensus client address was not a valid URL"))
		}
	}

	if b.Execution.Metrics.Peers.Enabled {
		if err := validateURL(b.Execution.Address); err != nil {
			return false, errors.Join(err, errors.New("execution client address was not a valid URL"))
		}
	}

	if b.SSV.Metrics.Peers.Enabled || b.SSV.Metrics.Connections.Enabled {
		if err := validateURL(b.SSV.Address); err != nil {
			return false, errors.Join(err, errors.New("SSV client address was not a valid URL"))
		}
	}

	network := network.Name(b.Network)
	if err := network.Validate(); err != nil {
		return false, errors.Join(err, errors.New("network name was not valid"))
	}

	return true, nil
}
