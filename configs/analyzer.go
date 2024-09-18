package configs

import (
	"errors"
	"strings"
)

type Analyzer struct {
	LogFilePath string   `mapstructure:"log-file-path"`
	Operators   []string `mapstructure:"operators"`
	Cluster     bool     `mapstructure:"cluster"`
}

func (a Analyzer) Validate() (bool, error) {
	if a.LogFilePath == "" {
		return false, errors.New("log file path was empty")
	}
	if strings.Contains(a.LogFilePath, "../") {
		return false, errors.New("❕ flag should not contain traversal")
	}

	return true, nil
}
