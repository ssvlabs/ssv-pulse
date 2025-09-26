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
	"Block arrived before 1/3 slot",
	"no attester or sync-committee duties to execute",
	"executing committee duty",
	"late duty execution",
	"starting duty processing",
	// TODO - this should eventually be replaced by the next step ("fetched attestation data from CL")
	"successfully fetched attestation data",
	"fetched attestation data from CL",
	"got pre consensus quorum",
	"starting QBFT instance",
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
			// Line containing unexpected error that mentions either target slot or duty ID is relevant.
			// TODO - committee-duty-id lacks slot number, thus we have to additionally filter by target-slot
			// in order to filter out the noise (duties for different slots). Once we add slot number to
			// committee-duty-id we should replace this condition with the one below (the commented out lines).
			if containsUnexpectedCommitteeError(line) && (targetSlot == phase0.Slot(0) && relevantForDutyID(line, dutyID) ||
				targetSlot != phase0.Slot(0) && relevantForSlot(line, targetSlot)) {
				return true
			}
			//if containsUnexpectedCommitteeError(line) && (relevantForDutyID(line, dutyID) || relevantForSlot(line, targetSlot)) {
			//	return true
			//}

			// Special lines are relevant only if the target slot has been specified.
			if specialCommitteeDutyLines(line) && relevantForSlot(line, targetSlot) {
				return true
			}

			// The line is interesting only if it references a specific duty-step, the rest would be noise.
			// TODO - committee-duty-id lacks slot number, thus we have to additionally filter by target-slot
			// in order to filter out the noise (duties for different slots). Once we add slot number
			// to committee-duty-id we should replace this condition with the one below (the commented out lines).
			if relevantCommitteeDutyStep(line) && (dutyID != "" && relevantForDutyID(line, dutyID)) && relevantForSlot(line, targetSlot) {
				return true
			}
			//if relevantCommitteeDutyStep(line) && (dutyID != "" && relevantForDutyID(line, dutyID) {
			//	return true
			//}

			// The line is interesting only if it references a specific duty-step, the rest would be noise.
			if relevantCommitteeDutyStep(line) && relevantForSlot(line, targetSlot) {
				return true
			}

			return false
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

func specialCommitteeDutyLines(line string) bool {
	// This is a special handling of legacy log-line (that contains "got duties").
	if strings.Contains(line, "got duties") && strings.Contains(line, "\"handler\":\"ATTESTER\"") {
		return true
	}
	// This is a special handling of legacy log-line (that contains "ticker event").
	if strings.Contains(line, "ticker event") && strings.Contains(line, "\"handler\":\"CLUSTER\"") {
		return true
	}

	// This is a special handling a duty-relevant log-line (that contains "Block arrived before 1/3 slot").
	if helper.ContainsCaseInsensitive(line, "Block arrived before 1/3 slot") {
		return true
	}

	return false
}

func relevantCommitteeDutyStep(line string) bool {
	if helper.ContainsCaseInsensitive(line, "SYNC_COMMITTEE") {
		return false
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
