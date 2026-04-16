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

var dutyStepsProposer = []string{
	"ticker event",
	"got duties",
	"executing validator duty",
	"late duty execution",
	"starting duty processing",
	"signed & broadcasted partial RANDAO signature",
	"got pre-consensus message",
	"got pre-consensus quorum",
	"waited out proposer delay",
	"got beacon block proposal",
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
	"got post-consensus message",
	"got post-consensus quorum",
	"submitting block proposal",
	"successfully submitted block proposal",
	"finished duty processing",
}

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

func (s *Proposer) Analyze(logFilePath string, dutyID string, targetSlot phase0.Slot) error {
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

		// While parsing, skip a bunch of lines at the start of the file (if necessary) as a work-around for
		// occasional junk that can be found there sometimes.
		entry, lineTrimmed, err := s.logParser.ParseLogLine(line)
		if err != nil {
			if lineNumber < 20 {
				continue // probably just some junk we need to skip
			}
			return fmt.Errorf("parse log line %d `%s`, err: %w", lineNumber, line, err)
		}

		// If the slot-number wasn't explicitly set, try to figure out what slot this line corresponds to
		// by parsing this log line matching against known patterns.
		if targetSlot == phase0.Slot(0) {
			targetSlot, err = helper.TryParseSlot(line)
			if err != nil {
				return fmt.Errorf("parse target slot: %w", err)
			}
		}

		timeIntoSlot, err := s.getTimeIntoSlot(targetSlot, entry.Timestamp)
		if err != nil {
			return fmt.Errorf("get time into slot for line %d `%s`, err: %w", lineNumber, line, err)
		}

		lineIsRelevant := func() bool {
			// Note, this condition uses maybeRelevantForSlot (and not relevantForSlot) to ensure we don't
			// miss any potentially relevant errors at the cost of getting occasional false-positives.
			if containsUnexpectedProposerError(lineTrimmed) &&
				(relevantForDutyID(lineTrimmed, dutyID) || maybeRelevantForSlot(lineTrimmed, targetSlot, timeIntoSlot)) {
				return true
			}

			if specialProposerDutyLines(lineTrimmed) &&
				(relevantForDutyID(lineTrimmed, dutyID) || relevantForSlot(lineTrimmed, targetSlot)) {
				return true
			}

			if relevantProposerDutyStep(lineTrimmed) &&
				(relevantForDutyID(lineTrimmed, dutyID) || relevantForSlot(lineTrimmed, targetSlot)) {
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

func (s *Proposer) getTimeIntoSlot(targetSlot phase0.Slot, lineTimestamp time.Time) (time.Duration, error) {
	if targetSlot == phase0.Slot(0) {
		return 0, nil
	}

	targetSlotStartTime, err := s.blockchain.SlotStartTime(targetSlot)
	if err != nil {
		return 0, fmt.Errorf("get target slot %d start time: %w", targetSlot, err)
	}

	return lineTimestamp.Sub(targetSlotStartTime), nil
}

// specialProposerDutyLines highlights certain duty-relevant log-lines that will be skipped (filtered out) by
// other rules we have defined.
func specialProposerDutyLines(line string) bool {
	if strings.Contains(line, "got duties") && strings.Contains(line, "\"handler\":\"PROPOSER\"") {
		return true
	}
	if strings.Contains(line, "ticker event") && strings.Contains(line, "\"handler\":\"PROPOSER\"") {
		return true
	}
	if helper.ContainsCaseInsensitive(line, "reorg") && strings.Contains(line, "\"handler\":\"PROPOSER\"") {
		return true
	}
	if strings.Contains(line, "received proposal") {
		return true
	}
	if strings.Contains(line, "selected best proposal") {
		return true
	}

	return false
}

func relevantProposerDutyStep(line string) bool {
	if !maybeRelevantForProposer(line) {
		return false
	}

	for _, dutyStep := range dutyStepsProposer {
		if strings.Contains(line, dutyStep) {
			return true
		}
	}

	return false
}

func maybeRelevantForProposer(line string) bool {
	return helper.ContainsCaseInsensitive(line, "proposer")
}
