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

type Proposer struct {
	blockchain *environment.Blockchain
	logParser  environment.LogParser
}

func NewProposer(blockchain *environment.Blockchain, logParser environment.LogParser) *Proposer {
	return &Proposer{
		blockchain: blockchain,
		logParser:  logParser,
	}
}

func (s *Proposer) AnalyzeLog(logFilePath string, targetSlot phase0.Slot) error {
	logFile, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	logger := slog.With("duty_type", dutyTypeProposerPattern)

	lineNumber := 0
	scanner := parser.NewScanner(logFile)
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		if !helper.ContainsCaseInsensitive(line, dutyTypeProposerPattern) {
			continue
		}
		targetSlotPattern := fmt.Sprintf(slotPattern, targetSlot)
		// TODO
		//if !strings.Contains(line, targetSlotPattern) {
		//	continue
		//}
		const vPubkey = "903dff3e6a2615754803e58e320d206056535c354c1b650793b0c14c00017de4fc341b25869928a83a3bcaa45f943379"
		if !strings.Contains(line, targetSlotPattern) && !strings.Contains(line, vPubkey) {
			continue
		}

		if containsUnexpectedProposerError(line) {
			if err := s.logWithTimeIntoSlot(logger, line, lineNumber, targetSlot); err != nil {
				return err
			}
		}
		for _, dutyStep := range dutyStepsProposer {
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

func (s *Proposer) logWithTimeIntoSlot(logger *slog.Logger, line string, lineNumber int, targetSlot phase0.Slot) error {
	targetSlotStartTime, err := s.blockchain.SlotStartTime(targetSlot)
	if err != nil {
		return fmt.Errorf("get target slot start time: %w", err)
	}

	entry, err := s.logParser.ParseLogLine(line)
	if err != nil {
		return fmt.Errorf("parse log line %d `%s`, err: %w", lineNumber, line, err)
	}
	timeIntoSlot := entry.Timestamp.Sub(targetSlotStartTime)

	timeIntoSlotStr := fmt.Sprintf("time_into_slot_ms=%d", timeIntoSlot.Milliseconds())
	logger.Info(timeIntoSlotStr + " " + line)

	return nil
}
