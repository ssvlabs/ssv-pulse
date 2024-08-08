package configs

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)


type Config struct {
	Network         NetworkName
	BeaconNodeAddrs []Address
}

func Init(addrFlag,networkFlag string ) (Config, error) {

	flag.Parse()

	var cfg Config
	addresses, err := parseAddresses(addrFlag)
	if err != nil {
		return cfg, err
	}
	for _, addr := range addresses {
		if err := addr.Validate(); err != nil {
			return cfg, errors.Join(err, errors.New("one of the addressses was not valid"))
		}
	}

	network := NetworkName(networkFlag)
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
			return addresses, fmt.Errorf("'%s' flag contains duplicates", "address")
		}
		addresses = append(addresses, Address(addr))
		seen[addr] = struct{}{}
	}
	return addresses, nil
}
