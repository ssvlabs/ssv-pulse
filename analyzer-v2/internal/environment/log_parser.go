package environment

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/araddon/dateparse"

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

	var standardLogEntry struct {
		Timestamp string `json:"time"`
	}

	err = json.Unmarshal([]byte(trimmedLine), &standardLogEntry)
	if err != nil {
		return LogEntry{}, trimmedLine, fmt.Errorf("unmarshal log line: %s, err %w", trimmedLine, err)
	}

	ts, err := dateparse.ParseAny(standardLogEntry.Timestamp)
	if err != nil {
		return LogEntry{}, trimmedLine, fmt.Errorf("external log entry: parse timestamp: %s: %w", standardLogEntry.Timestamp, err)
	}

	// TODO - no longer need this ? Remove then.
	// trim certain log-lines (those listing duties) to keep the output reasonably short
	//trimmedLine = helper.TrimRightOf(line, "\"duties\"") + " ..."

	return LogEntry{
		Timestamp: ts,
	}, trimmedLine, nil
}

// ExternalLogParser is a log parser that understands log format used by external SSV Operators (managed
// by 3rd-party entities, not ssv-labs).
type ExternalLogParser struct{}

func (parser *ExternalLogParser) ParseLogLine(line string) (entry LogEntry, trimmedLine string, err error) {
	// nothing to trim for production format
	trimmedLine = line

	var externalLogEntry struct {
		Timestamp string `json:"T"`
	}

	err = json.Unmarshal([]byte(trimmedLine), &externalLogEntry)
	if err != nil {
		return LogEntry{}, trimmedLine, fmt.Errorf("unmarshal log line: %s, err %w", trimmedLine, err)
	}

	ts, err := dateparse.ParseAny(externalLogEntry.Timestamp)
	if err != nil {
		return LogEntry{}, trimmedLine, fmt.Errorf("external log entry: parse timestamp: %s: %w", externalLogEntry.Timestamp, err)
	}

	return LogEntry{
		Timestamp: ts,
	}, trimmedLine, nil
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
