package configs

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var Values Config

type Config struct {
	Benchmark Benchmark `mapstructure:"benchmark"`
	Analyzer  Analyzer  `mapstructure:"analyzer"`
}

func sanitizeURL(str string) (string, error) {
	parsedURL, err := url.Parse(str)
	if err != nil {
		return "", err
	}
	var validationErr error
	if parsedURL.Scheme == "" {
		validationErr = errors.Join(validationErr, errors.New("scheme was empty"))
	}
	if parsedURL.Host == "" {
		validationErr = errors.Join(validationErr, errors.New("host was empty"))
	}

	if validationErr != nil {
		return "", validationErr
	}

	return strings.TrimSuffix(fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.EscapedPath()), "/"), nil
}
