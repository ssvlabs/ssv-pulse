package environment

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ssvlabs/ssv-pulse/analyzer-v2/internal/helper"
)

type LogParser interface {
	ParseLogLine(line string) (LogEntry, error)
}

type LogEntry struct {
	Timestamp time.Time
}

type StandardLogParser struct{}

func (parser *StandardLogParser) ParseLogLine(line string) (LogEntry, error) {
	// trim additional info (such as timestamps) added by environment logger itself
	line = helper.TrimLeftOf(line, "{")

	var entry standardLogEntry
	err := json.Unmarshal([]byte(line), &entry)
	if err != nil {
		return LogEntry{}, fmt.Errorf("unmarshal log line: %s, err %w", line, err)
	}
	return LogEntry{
		Timestamp: entry.Timestamp,
	}, nil
}

type standardLogEntry struct {
	Timestamp time.Time `json:"time"`
}

type KrakenLogParser struct{}

func (parser *KrakenLogParser) ParseLogLine(line string) (LogEntry, error) {
	var entry krakenLogEntry
	err := json.Unmarshal([]byte(line), &entry)
	if err != nil {
		return LogEntry{}, fmt.Errorf("unmarshal log line: %s, err %w", line, err)
	}
	return LogEntry{
		Timestamp: entry.Timestamp,
	}, nil
}

type krakenLogEntry struct {
	Timestamp time.Time `json:"T"`
}

func LogParserByName(name string) (LogParser, error) {
	if name == "standard" {
		return &StandardLogParser{}, nil
	}
	if name == "kraken" {
		return &KrakenLogParser{}, nil
	}
	return nil, fmt.Errorf("unknown logs parser: %s", name)
}
