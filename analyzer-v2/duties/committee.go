package duties

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

const dutyTypeCommitteePattern = "committee"
const slotPattern = "\"slot\":%d"

type Committee struct {
	dutySteps []string
}

func NewCommittee() *Committee {
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
	}
}

func (s *Committee) AnalyzeLog(logFilePath string, targetSlot uint64) error {
	logFile, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	logger := slog.With("duty_type", dutyTypeCommitteePattern)

	scanner := parser.NewScanner(logFile)

	lineNumber := 0
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
				logger.Info(line)

				// TODO
				//var entry commitLogEntry
				//if err := json.Unmarshal([]byte(line), &entry); err != nil {
				//	return fmt.Errorf("unmarshal log line %d (file = `%s`): `%s`, err: %w", lineNumber, logFile.Name(), line, err)
				//}
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

// TODO - need this ?
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
