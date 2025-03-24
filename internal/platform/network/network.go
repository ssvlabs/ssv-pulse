package network

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

type (
	Name    string
	Network struct {
		Name        Name
		GenesisTime time.Time
	}
)

const (
	Holesky Name = "holesky"
	Mainnet Name = "mainnet"
	Hoodi   Name = "hoodi"
)

var (
	Supported = map[Name]Network{
		Holesky: {Name: Holesky, GenesisTime: time.Unix(1695902400, 0)},
		Mainnet: {Name: Mainnet, GenesisTime: time.Unix(1606824023, 0)},
		Hoodi:   {Name: Hoodi, GenesisTime: time.Unix(1742213400, 0)},
	}
)

func (n Name) Validate() error {
	_, ok := Supported[Name(strings.ToLower(string(n)))]
	if !ok {
		return fmt.Errorf("unsupported network name: '%s'. List of supported networks: '%v'", n, slices.Collect(maps.Keys(Supported)))
	}

	return nil
}
