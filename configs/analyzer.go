package configs

import (
	"errors"
	"strings"
)

type Analyzer struct {
	LogFilePath string   `mapstructure:"log-file-path"`
	Operators   []uint32 `mapstructure:"operators"`
	Cluster     bool     `mapstructure:"cluster"`
}

func (a Analyzer) Validate() (bool, error) {
	if a.LogFilePath == "" {
		return false, errors.New("log file path was empty")
	}
	if strings.Contains(a.LogFilePath, "../") {
		return false, errors.New("❕ flag should not contain traversal")
	}

	if a.Cluster && len(a.Operators) == 0 {
		return false, errors.New("if cluster is set to 'true', the list of operators cannot be empty")
	}

	return true, nil
}

func (a Analyzer) WithScores() bool {
	return len(a.Operators) != 0 && a.Cluster
}
