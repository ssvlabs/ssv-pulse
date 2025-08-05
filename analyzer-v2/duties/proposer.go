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
