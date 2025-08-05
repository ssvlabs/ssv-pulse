package config

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type Config struct {
	LogFilesDirectory string      `mapstructure:"log-files-directory"`
	Blockchain        string      `mapstructure:"blockchain"`
	LogParser         string      `mapstructure:"log-parser"`
	TargetSlot        phase0.Slot `mapstructure:"target-slot"`
}

func (c *Config) Validate() error {
	if c.LogFilesDirectory == "" {
		return fmt.Errorf("❕ 'log-files-directory' was not specified")
	}
	if c.Blockchain == "" {
		return fmt.Errorf("❕ 'blockchain' was not specified")
	}
	if c.LogParser == "" {
		return fmt.Errorf("❕ 'log-parser' was not specified")
	}
	if c.TargetSlot == 0 {
		return fmt.Errorf("❕ 'target-slot' was not specified")
	}

	return nil
}
