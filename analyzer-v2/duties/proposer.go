package duties

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

type Proposer struct {
}

func NewProposer() *Proposer {
	return &Proposer{}
}

func (s *Proposer) Analyze(logFilePath string) error {
	const dutyType = "proposer"

	const (
		dutyStartPattern = "starting duty processing"
		dutyEndPattern   = "TODO"
	)

	logFile, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	logger := slog.With("duty_type", dutyType)

	scanner := parser.NewScanner(logFile)

	lineNumber := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		if !strings.Contains(line, dutyType) {
			continue
		}
		if strings.Contains(line, dutyStartPattern) {
			logger.Info(line)

			// TODO
			//var entry commitLogEntry
			//if err := json.Unmarshal([]byte(line), &entry); err != nil {
			//	return fmt.Errorf("unmarshal log line %d (file = `%s`): `%s`, err: %w", lineNumber, logFile.Name(), line, err)
			//}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// TODO
//type commitLogEntry struct {
//	Timestamp     parser.MultiFormatTime `json:"T"`
//	Round         uint8                  `json:"round"`
//	DutyID        string                 `json:"duty_id"`
//	Message       string                 `json:"M"`
//	CommitSigners []parser.SignerID      `json:"commit_signers"` //NOTE: This array always contains 1 item
//}
//
//func (p *commitLogEntry) UnmarshalJSON(data []byte) error {
//	type Entry commitLogEntry
//
//	alias := &struct {
//		CommitSignersDash []parser.SignerID `json:"commit-signers"`
//		*Entry
//	}{
//		Entry: (*Entry)(p),
//	}
//
//	if err := json.Unmarshal(data, &alias); err != nil {
//		return err
//	}
//
//	if alias.CommitSignersDash != nil {
//		p.CommitSigners = alias.CommitSignersDash
//	}
//
//	return nil
//}
