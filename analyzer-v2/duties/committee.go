package duties

import (
	"fmt"
	"os"
	"strings"
	"time"

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
	"got pre-consensus message",
	"got pre-consensus quorum",
	"starting QBFT instance",
	"leader broadcasting proposal message",
	"got proposal message",
	"got prepare message",
	"got prepare quorum",
	"got commit message",
	"got commit quorum",
	"round timed out",
	"got round change",
	"got justified round change",
	"QBFT instance is decided",
	"broadcasted post-consensus partial signature message",
	// TODO - this should eventually be replaced by the next step ("got post-consensus message")
	"got partial signatures",
	"got post-consensus message",
	"got post-consensus quorum",
	"submitting attestations",
	"successfully submitted attestations",
	"submitting sync committee",
	"successfully submitted sync committee",
	"finished duty processing",
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

	lineNumber := 0
	scanner := parser.NewScanner(logFile)
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		// If the slot-number wasn't explicitly set, try to figure out what slot this line corresponds to
		// by parsing this log line matching against known patterns.
		slot := targetSlot
		if slot == phase0.Slot(0) {
			slot = helper.TryParseSlot(line)
		}

		// While parsing, skip a bunch of lines at the start of the file (if necessary) as a work-around for
		// occasional junk that can be found there sometimes.
		entry, lineTrimmed, err := s.logParser.ParseLogLine(line)
		if err != nil {
			if lineNumber < 20 {
				continue // probably just some junk we need to skip
			}
			return fmt.Errorf("parse log line %d `%s`, err: %w", lineNumber, line, err)
		}

		timeIntoSlot, err := s.getTimeIntoSlot(slot, entry.Timestamp)
		if err != nil {
			return fmt.Errorf("get time into slot for line %d `%s`, err: %w", lineNumber, line, err)
		}

		lineIsRelevant := func() bool {
			//// Line containing unexpected error that mentions either target slot or duty ID is relevant.
			//// TODO - committee-duty-id lacks slot number, thus we have to additionally filter by target-slot
			//// in order to filter out the noise (duties for different slots). Once we add slot number to
			//// committee-duty-id we should replace this condition with the one below (the commented out lines).
			//if containsUnexpectedCommitteeError(lineTrimmed) && (slot == phase0.Slot(0) && relevantForDutyID(lineTrimmed, dutyID) ||
			//	slot != phase0.Slot(0) && relevantForSlot(lineTrimmed, slot, timeIntoSlot) {
			//	return true
			//}
			if containsUnexpectedCommitteeError(lineTrimmed) &&
				(relevantForDutyID(lineTrimmed, dutyID) || maybeRelevantForSlot(lineTrimmed, slot, timeIntoSlot)) {
				return true
			}

			if specialCommitteeDutyLines(lineTrimmed) && relevantForSlot(lineTrimmed, slot) {
				return true
			}

			//// TODO - committee-duty-id lacks slot number, thus we have to additionally filter by target-slot
			//// in order to filter out the noise (duties for different slots). Once we add slot number
			//// to committee-duty-id we should replace this condition with the one below (the commented out lines).
			//if relevantCommitteeDutyStep(lineTrimmed) && relevantForDutyID(lineTrimmed, dutyID) &&
			//   relevantForSlot(lineTrimmed, slot, timeIntoSlot) {
			//	return true
			//}
			if relevantCommitteeDutyStep(lineTrimmed) && relevantForDutyID(lineTrimmed, dutyID) {
				return true
			}

			if relevantCommitteeDutyStep(lineTrimmed) && relevantForSlot(lineTrimmed, slot) {
				return true
			}

			return false
		}()

		if !lineIsRelevant {
			continue
		}

		fmt.Println(fmt.Sprintf("time_into_slot_ms=%d", timeIntoSlot.Milliseconds()) + " " + lineTrimmed)
	}
	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("read %d log lines, scanner error: %w", lineNumber, err)
	}

	return nil
}

func (s *Committee) getTimeIntoSlot(targetSlot phase0.Slot, lineTimestamp time.Time) (time.Duration, error) {
	if targetSlot == phase0.Slot(0) {
		return 0, nil
	}

	targetSlotStartTime, err := s.blockchain.SlotStartTime(targetSlot)
	if err != nil {
		return 0, fmt.Errorf("get target slot start time: %w", err)
	}

	return lineTimestamp.Sub(targetSlotStartTime), nil
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
