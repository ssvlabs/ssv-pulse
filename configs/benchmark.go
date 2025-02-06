package configs

import (
	"errors"
	"net/url"
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
	Peers   Metric `mapstructure:"peers"`
	Latency Metric `mapstructure:"latency"`
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
	Addresses []string         `mapstructure:"address"`
	Metrics   ConsensusMetrics `mapstructure:"metrics"`
}

func (c Consensus) AddrURLs() ([]*url.URL, error) {
	var parsedURLs []*url.URL
	for _, address := range c.Addresses {
		parsedURL, err := url.Parse(address)
		if err != nil {
			return nil, errors.Join(err, errors.New("error parsing Consensus address to URL type"))
		}
		parsedURLs = append(parsedURLs, parsedURL)
	}

	return parsedURLs, nil
}

type Execution struct {
	Addresses []string         `mapstructure:"address"`
	Metrics   ExecutionMetrics `mapstructure:"metrics"`
}

func (e Execution) AddrURLs() ([]*url.URL, error) {
	var urls []*url.URL
	for _, addr := range e.Addresses {
		parsedURL, err := url.Parse(addr)
		if err != nil {
			return nil, errors.Join(err, errors.New("error parsing Execution address to URL type"))
		}
		urls = append(urls, parsedURL)
	}

	return urls, nil
}

type SSV struct {
	Address string     `mapstructure:"address"`
	Metrics SSVMetrics `mapstructure:"metrics"`
}

func (s SSV) AddrURL() (*url.URL, error) {
	parsedURL, err := url.Parse(s.Address)
	if err != nil {
		return nil, errors.Join(err, errors.New("error parsing SSV address to URL type"))
	}
	return parsedURL, nil
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

func (b *Benchmark) Validate() (bool, error) {
	if b.Consensus.Metrics.Peers.Enabled ||
		b.Consensus.Metrics.Attestation.Enabled ||
		b.Consensus.Metrics.Client.Enabled ||
		b.Consensus.Metrics.Latency.Enabled {
		var urls []string
		for _, addr := range b.Consensus.Addresses {
			url, err := sanitizeURL(addr)
			if err != nil {
				return false, errors.Join(err, errors.New("consensus client address was not a valid URL"))
			}
			urls = append(urls, url)
		}

		b.Consensus.Addresses = urls
	}

	if b.Execution.Metrics.Peers.Enabled || b.Execution.Metrics.Latency.Enabled {
		var urls []string
		for _, addr := range b.Execution.Addresses {
			url, err := sanitizeURL(addr)
			if err != nil {
				return false, errors.Join(err, errors.New("execution client address was not a valid URL"))
			}
			urls = append(urls, url)
		}

		b.Execution.Addresses = urls
	}

	if b.SSV.Metrics.Peers.Enabled || b.SSV.Metrics.Connections.Enabled {
		url, err := sanitizeURL(b.SSV.Address)
		if err != nil {
			return false, errors.Join(err, errors.New("SSV client address was not a valid URL"))
		}
		b.SSV.Address = url
	}

	network := network.Name(b.Network)
	if err := network.Validate(); err != nil {
		return false, errors.Join(err, errors.New("network name was not valid"))
	}

	return true, nil
}
