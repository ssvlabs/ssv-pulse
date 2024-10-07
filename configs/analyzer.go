package configs

import (
	"errors"
	"fmt"
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

	if a.Cluster {
		if !isValidClusterSize(a.Operators) {
			return false, fmt.Errorf("the cluster size: '%d' is not valid'", len(a.Operators))
		}
	}

	return true, nil
}

func (a Analyzer) WithScores() bool {
	return len(a.Operators) != 0 && a.Cluster
}

func isValidClusterSize(operators []uint32) bool {
	if len(operators) < 4 || len(operators) > 13 || len(operators)%3 != 1 {
		return false
	}

	return true
}
