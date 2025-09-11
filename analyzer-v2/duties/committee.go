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

var dutyStepsCommittee = []string{
	"ticker event",
	"got duties",
	"no attester or sync-committee duties to execute",
	"executing committee duty",
	"late duty execution",
	"starting duty processing",
	// TODO - this should eventually be replaced by the next step ("fetched attestation data from CL")
	"successfully fetched attestation data",
	"fetched attestation data from CL",
	"got pre consensus quorum",
	// TODO - this should eventually be replaced by the next step ("fetched attestation data from CL")
	"starting QBFT instance",
	"starting new QBFT instance",
	"leader broadcasting proposal message",
	"got proposal message",
	"got prepare message",
	"got prepare quorum",
	"got prepare quorum",
	"got commit quorum",
	"round timed out",
	"got round change",
	"got justified round change",
	"QBFT instance is decided",
	"constructed & signed post consensus partial signature message",
	"broadcasted post consensus partial signature message",
	"got partial signatures",
	"got post consensus quorum",
	"submitting attestations",
	"successfully submitted attestations",
	"submitting sync committee",
	"successfully submitted sync committee",
	"successfully finished duty processing",
}

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

func (s *Committee) Analyze(logFilePath string, dutyID string, targetSlot phase0.Slot) error {
	logFile, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	logger := slog.With("duty_type", "committee")

	lineNumber := 0
	scanner := parser.NewScanner(logFile)
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		lineIsRelevant := func() bool {
			// Log-line must be either relevant to the target-slot or the specified duty-id, otherwise it's noise.
			if !(targetSlot != 0 && relevantForSlot(line, targetSlot)) && !(dutyID != "" && relevantForDutyID(line, dutyID)) {
				return false
			}

			return relevantForCommitteeDuty(line)
		}()

		if !lineIsRelevant {
			continue
		}

		if err := s.logWithTimeIntoSlot(logger, line, lineNumber, targetSlot); err != nil {
			return err
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

func relevantForCommitteeDuty(line string) bool {
	if containsUnexpectedCommitteeError(line) {
		return true
	}

	// Clean up the line from false-positive triggers it potentially might have.
	line = strings.ReplaceAll(line, "\"committee_index\":", "")
	line = strings.ReplaceAll(line, "\"handler\":\"SYNC_COMMITTEE\"", "")

	// This is a special handling of legacy log-line (that contains "ticker event").
	if strings.Contains(line, "ticker event") && strings.Contains(line, "\"handler\":\"CLUSTER\"") {
		return true
	}
	// This is a special handling of legacy log-line (that contains "got duties").
	if strings.Contains(line, "got duties") && strings.Contains(line, "\"handler\":\"ATTESTER\"") {
		return true
	}

	if !maybeRelevantForCommittee(line) {
		return false
	}

	for _, dutyStep := range dutyStepsCommittee {
		if strings.Contains(line, dutyStep) {
			return true
		}
	}

	return false
}

func maybeRelevantForCommittee(line string) bool {
	return helper.ContainsCaseInsensitive(line, "committee")
}
