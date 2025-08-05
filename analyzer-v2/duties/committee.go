package duties

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ssvlabs/ssv-pulse/analyzer-v2/internal/environment"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

const dutyTypeCommitteePattern = "committee"
const slotPattern = "\"slot\":%d"

type Committee struct {
	dutySteps []string

	blockchain *environment.Blockchain
	logParser  environment.LogParser
}

func NewCommittee(blockchain *environment.Blockchain, logParser environment.LogParser) *Committee {
	return &Committee{
		dutySteps: []string{
			"starting duty processing",
			"fetched attestation data from CL",
			"QBFT instance decided",
			"constructed & signed post consensus partial signature message",
			"broadcasted post consensus partial signature message",
			"got post consensus quorum",
			"submitting attestations",
			"successfully submitted attestations",
			"submitting sync committee",
			"successfully submitted sync committee",
			"successfully finished duty processing",
		},
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

	targetSlotStartTime, err := s.blockchain.SlotStartTime(targetSlot)
	if err != nil {
		return fmt.Errorf("get target slot start time: %w", err)
	}

	logger := slog.With("duty_type", dutyTypeCommitteePattern)

	lineNumber := 0
	scanner := parser.NewScanner(logFile)
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		if !strings.Contains(line, dutyTypeCommitteePattern) {
			continue
		}
		targetSlotPattern := fmt.Sprintf(slotPattern, targetSlot)
		if !strings.Contains(line, targetSlotPattern) {
			continue
		}

		for _, dutyStep := range s.dutySteps {
			if strings.Contains(line, dutyStep) {
				entry, err := s.logParser.ParseLogLine(line)
				if err != nil {
					return fmt.Errorf("parse log line %d `%s`, err: %w", lineNumber, line, err)
				}

				timeIntoSlot := entry.Timestamp.Sub(targetSlotStartTime)

				logger.With("time_into_slot_ms", timeIntoSlot.Milliseconds()).Info(line)
			}
		}
	}
	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("read %d log lines, scanner error: %w", lineNumber, err)
	}

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
