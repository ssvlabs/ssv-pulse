package config

import (
	"errors"

	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/network"
)

func IsValid(consensusAddr, executionAddr, ssvAddr string, networkFlag string) (bool, error) {
	if consensusAddr == "" {
		return false, errors.New("consensus client address was not valid")
	}

	if executionAddr == "" {
		return false, errors.New("execution client address was not valid")
	}

	if ssvAddr == "" {
		return false, errors.New("execution client address was not valid")
	}

	network := network.Name(networkFlag)
	if err := network.Validate(); err != nil {
		return false, errors.Join(err, errors.New("network name was not valid"))
	}
	return true, nil
}
