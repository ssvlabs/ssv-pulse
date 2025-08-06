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

func (s *Committee) AnalyzeLog(logFilePath string, targetSlot phase0.Slot) error {
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

	entry, err := s.logParser.ParseLogLine(line)
	if err != nil {
		return fmt.Errorf("parse log line %d `%s`, err: %w", lineNumber, line, err)
	}
	timeIntoSlot := entry.Timestamp.Sub(targetSlotStartTime)

	timeIntoSlotStr := fmt.Sprintf("time_into_slot_ms=%d", timeIntoSlot.Milliseconds())
	logger.Info(timeIntoSlotStr + " " + line)

	return nil
}

// TODO - need this ?
////func (s *Committee) AnalyzeJson(logFilePath string) error {
////	jsonFile, err := os.Open(logFilePath)
////	if err != nil {
////		return fmt.Errorf("open json file: %w", err)
////	}
////	defer func() {
////		_ = jsonFile.Close()
////	}()
////
////	logger := slog.With("duty_type", dutyTypeCommitteePattern)
////
////	jsonFileContents, err := io.ReadAll(jsonFile)
////	if err != nil {
////		return fmt.Errorf("read json file contents: `%s`, err: %w", jsonFile.Name(), err)
////	}
////	var entries []jsonEntry
////	err = json.Unmarshal(jsonFileContents, &entries)
////	if err != nil {
////		return fmt.Errorf("parse json file contents: `%s`, err: %w", jsonFile.Name(), err)
////	}
////
////	lineNumber := 0
////	for _, entry := range entries {
////		line := entry.Line
////		lineNumber++
////
////		if !strings.Contains(line, dutyTypeCommitteePattern) {
////			continue
////		}
////
////		for _, dutyStep := range s.dutySteps {
////			if strings.Contains(line, dutyStep) {
////				logger.Info(line)
////
////				// TODO
////				//var entry commitLogEntry
////				//if err := json.Unmarshal([]byte(line), &entry); err != nil {
////				//	return fmt.Errorf("unmarshal log line %d (file = `%s`): `%s`, err: %w", lineNumber, logFile.Name(), line, err)
////				//}
////			}
////		}
////	}
////
////	return nil
////}
//
//type jsonEntry struct {
//	Line string `json:"line"`
//}
