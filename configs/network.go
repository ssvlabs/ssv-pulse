package configs

import (
	"fmt"
	"strings"
	"time"
)

var (
	GenesisTime = map[NetworkName]time.Time{
		Holesky: time.Unix(1695902400, 0),
		Mainnet: time.Unix(1606824023, 0),
	}
)

type NetworkName string

const (
	Holesky NetworkName = "holesky"
	Mainnet NetworkName = "mainnet"
)

func (n NetworkName) Validate() error {
	if !strings.EqualFold(string(n), string(Holesky)) && !strings.EqualFold(string(n), string(Mainnet)) {
		return fmt.Errorf("network name should be either '%s' or '%s'", Holesky, Mainnet)
	}
	return nil
}
