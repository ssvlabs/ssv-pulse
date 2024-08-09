package configs

import (
	"errors"
)

type Config struct {
	Network            NetworkName
	ConsensusNodeAddrs Address
	ExecutionNodeAddrs Address
	SSVNodeAddrs       Address
}

func Init(consensusAddr, executionAddr, ssvAddr Address, networkFlag string) (Config, error) {
	var cfg Config
	if err := consensusAddr.Validate(); err != nil {
		return cfg, errors.Join(err, errors.New("consensus client address was not valid"))
	}

	if err := executionAddr.Validate(); err != nil {
		return cfg, errors.Join(err, errors.New("execution client address was not valid"))
	}

	if err := ssvAddr.Validate(); err != nil {
		return cfg, errors.Join(err, errors.New("ssv client address was not valid"))
	}

	network := NetworkName(networkFlag)
	if err := network.Validate(); err != nil {
		return cfg, errors.Join(err, errors.New("network name was not valid"))
	}

	return Config{
		ConsensusNodeAddrs: consensusAddr,
		ExecutionNodeAddrs: executionAddr,
		SSVNodeAddrs:       ssvAddr,
		Network:            network,
	}, nil
}
