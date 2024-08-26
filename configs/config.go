package configs

import (
	"errors"
	"net/url"
)

var Values Config

type Config struct {
	Benchmark Benchmark `mapstructure:"benchmark"`
	Analyzer  Analyzer  `mapstructure:"analyzer"`
}

func validateURL(str string) error {
	parsedURL, err := url.Parse(str)
	if err != nil {
		return err
	}

	if parsedURL.Scheme == "" {
		return errors.New("scheme was empty")
	}

	if parsedURL.Host == "" {
		return errors.New("host was empty")
	}

	return nil
}
