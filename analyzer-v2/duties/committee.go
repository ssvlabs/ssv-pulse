package duties

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ssvlabs/ssv-pulse/analyzer-v2/internal/environment"
	"github.com/ssvlabs/ssv-pulse/analyzer-v2/internal/helper"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

type Committee struct {
	blockchain *environment.Blockchain
	logParser  environment.LogParser
}

func NewCommittee(blockchain *environment.Blockchain, logParser environment.LogParser) *Committee {
	return &Committee{
		blockchain: blockchain,
		logParser:  logParser,
	}
}

func (s *Committee) Analyze(logFilePath string, targetSlot phase0.Slot) error {
	logFile, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	logger := slog.With("duty_type", dutyTypeCommitteePattern)

	lineNumber := 0
	scanner := parser.NewScanner(logFile)
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		if !helper.ContainsCaseInsensitive(line, dutyTypeCommitteePattern) {
			continue
		}
		// since sync-committee-contribution cannot be filtered out by `dutyTypeCommitteePattern`
		// we have to do this additional filtering here
		if helper.ContainsCaseInsensitive(line, dutyTypeSyncCommitteePattern) {
			continue
		}

		targetSlotPattern := fmt.Sprintf(slotPattern, targetSlot)
		if !strings.Contains(line, targetSlotPattern) {
			continue
		}

		if containsUnexpectedCommitteeError(line) {
			if err := s.logWithTimeIntoSlot(logger, line, lineNumber, targetSlot); err != nil {
				return err
			}
		}
		for _, dutyStep := range dutyStepsCommittee {
			if strings.Contains(line, dutyStep) {
				if err := s.logWithTimeIntoSlot(logger, line, lineNumber, targetSlot); err != nil {
					return err
				}
			}
		}
	}
	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("read %d log lines, scanner error: %w", lineNumber, err)
	}

	return nil
}

func (s *Committee) logWithTimeIntoSlot(logger *slog.Logger, line string, lineNumber int, targetSlot phase0.Slot) error {
	targetSlotStartTime, err := s.blockchain.SlotStartTime(targetSlot)
	if err != nil {
		return fmt.Errorf("get target slot start time: %w", err)
	}

	entry, trimmedLine, err := s.logParser.ParseLogLine(line)
	if err != nil {
		return fmt.Errorf("parse log line %d `%s`, err: %w", lineNumber, line, err)
	}
	timeIntoSlot := entry.Timestamp.Sub(targetSlotStartTime)

	timeIntoSlotStr := fmt.Sprintf("time_into_slot_ms=%d", timeIntoSlot.Milliseconds())
	logger.Info(timeIntoSlotStr + " " + trimmedLine)

	return nil
}
