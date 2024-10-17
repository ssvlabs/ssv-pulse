package configs

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ssvlabs/ssv-pulse/internal/ssv"
)

type Analyzer struct {
	LogFilesDirectory string   `mapstructure:"log-files-directory"`
	Operators         []uint32 `mapstructure:"operators"`
	Cluster           bool     `mapstructure:"cluster"`
}

func (a Analyzer) Validate() (bool, error) {
	if a.LogFilesDirectory == "" {
		return false, errors.New("❕ log files directory was empty")
	}
	if strings.Contains(a.LogFilesDirectory, "../") {
		return false, errors.New("❕ flag should not contain traversal")
	}

	if a.Cluster && len(a.Operators) == 0 {
		return false, errors.New("❕ if cluster is set to 'true', the list of operators cannot be empty")
	}

	if a.Cluster {
		if !ssv.IsValidClusterSize(a.Operators) {
			return false, fmt.Errorf("the cluster size: '%d' is not valid'", len(a.Operators))
		}
	}

	return true, nil
}
