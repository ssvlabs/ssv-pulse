package configs

import "errors"

type Analyzer struct {
	LogFilePath string `mapstructure:"log-file-path"`
}

func (a Analyzer) Validate() (bool, error) {
	if a.LogFilePath == "" {
		return false, errors.New("log file path was empty")
	}

	return true, nil
}
