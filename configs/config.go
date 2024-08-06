package configs

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

const (
	addrFlagName    = "addresses"
	networkFlagName = "network"
)

type Config struct {
	Network         NetworkName
	BeaconNodeAddrs []Address
}

func Init() (Config, error) {
	addrFlag := flag.String(addrFlagName, "", "Comma-separated list of urls, e.g. 'http://eth2-lh-mainnet-5052.bloxinfra.com,http://mainnet-standalone-v3.bloxinfra.com:5052'")
	networkFlag := flag.String(networkFlagName, "", "Network to use, either 'mainnet' or 'holesky'")
	flag.Parse()

	var cfg Config
	addresses, err := parseAddresses(*addrFlag)
	if err != nil {
		return cfg, err
	}
	for _, addr := range addresses {
		if err := addr.Validate(); err != nil {
			return cfg, errors.Join(err, errors.New("one of the addressses was not valid"))
		}
	}

	network := NetworkName(*networkFlag)
	if err := network.Validate(); err != nil {
		return cfg, errors.Join(err, errors.New("network name was not valid"))
	}

	return Config{
		BeaconNodeAddrs: addresses,
		Network:         network,
	}, nil
}

func parseAddresses(addrFlag string) ([]Address, error) {
	seen := make(map[string]struct{})
	var addresses []Address

	pairs := strings.Split(addrFlag, ",")

	for _, addr := range pairs {
		if _, ok := seen[addr]; ok {
			return addresses, fmt.Errorf("'%s' flag contains duplicates", addrFlagName)
		}
		addresses = append(addresses, Address(addr))
		seen[addr] = struct{}{}
	}
	return addresses, nil
}
