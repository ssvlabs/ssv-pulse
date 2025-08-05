package config

import (
	"fmt"
)

type Config struct {
	LogFilesDirectory string `mapstructure:"log-files-directory"`
	TargetSlot        uint64 `mapstructure:"target-slot"`
}

func (c *Config) Validate() error {
	if c.LogFilesDirectory == "" {
		return fmt.Errorf("❕ 'log-files-directory' was not set in configuration")
	}
	if c.TargetSlot == 0 {
		return fmt.Errorf("❕ 'target-slot' was not set in configuration")
	}

	return nil
}
