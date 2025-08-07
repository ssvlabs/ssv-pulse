package environment

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ssvlabs/ssv-pulse/analyzer-v2/internal/helper"
)

type LogParser interface {
	ParseLogLine(line string) (entry LogEntry, trimmedLine string, err error)
}

type LogEntry struct {
	Timestamp time.Time
}

// StandardLogParser is a log parser that understands log format used by ssv-labs.
type StandardLogParser struct{}

func (parser *StandardLogParser) ParseLogLine(line string) (entry LogEntry, trimmedLine string, err error) {
	// trim additional info (such as timestamps) added by environment logger itself
	trimmedLine = helper.TrimLeftOf(line, "{")

	var e standardLogEntry
	err = json.Unmarshal([]byte(trimmedLine), &e)
	if err != nil {
		return LogEntry{}, trimmedLine, fmt.Errorf("unmarshal log line: %s, err %w", trimmedLine, err)
	}
	return LogEntry{
		Timestamp: e.Timestamp,
	}, trimmedLine, nil
}

type standardLogEntry struct {
	Timestamp time.Time `json:"time"`
}

// ExternalLogParser is a log parser that understands log format used by external SSV Operators (managed
// by 3rd-party entities, not ssv-labs).
type ExternalLogParser struct{}

func (parser *ExternalLogParser) ParseLogLine(line string) (entry LogEntry, trimmedLine string, err error) {
	// nothing to trim for production format
	trimmedLine = line

	var e krakenLogEntry
	err = json.Unmarshal([]byte(trimmedLine), &e)
	if err != nil {
		return LogEntry{}, trimmedLine, fmt.Errorf("unmarshal log line: %s, err %w", trimmedLine, err)
	}
	return LogEntry{
		Timestamp: e.Timestamp,
	}, trimmedLine, nil
}

type krakenLogEntry struct {
	Timestamp time.Time `json:"T"`
}

func LogParserByName(name string) (LogParser, error) {
	if name == "standard" {
		return &StandardLogParser{}, nil
	}
	if name == "external" {
		return &ExternalLogParser{}, nil
	}
	return nil, fmt.Errorf("unknown log format: %s", name)
}
