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

var dutyStepsSyncCommitteeContribution = []string{
	"ticker event",
	"got duties",
	"executing validator duty",
	"late duty execution",
	"starting duty processing",
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
	"got post-consensus message",
	"got post-consensus quorum",
	"submitting sync committee contributions",
	"successfully submitted sync committee contributions",
	"finished duty processing",
}

type SyncCommitteeContribution struct {
	blockchain *environment.Blockchain
	logParser  environment.LogParser
}

func NewSyncCommitteeContribution(blockchain *environment.Blockchain, logParser environment.LogParser) *SyncCommitteeContribution {
	return &SyncCommitteeContribution{
		blockchain: blockchain,
		logParser:  logParser,
	}
}

func (s *SyncCommitteeContribution) Analyze(logFilePath string, dutyID string, targetSlot phase0.Slot) error {
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
			if containsUnexpectedSyncCommitteeContributionError(lineTrimmed) &&
				(relevantForDutyID(lineTrimmed, dutyID) || maybeRelevantForSlot(lineTrimmed, slot, timeIntoSlot)) {
				return true
			}

			if specialSyncCommitteeContributionDutyLines(lineTrimmed) && relevantForSlot(lineTrimmed, slot) {
				return true
			}

			if relevantSyncCommitteeContributionDutyStep(lineTrimmed) && relevantForDutyID(lineTrimmed, dutyID) {
				return true
			}

			if relevantSyncCommitteeContributionDutyStep(lineTrimmed) && relevantForSlot(lineTrimmed, slot) {
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

func (s *SyncCommitteeContribution) getTimeIntoSlot(targetSlot phase0.Slot, lineTimestamp time.Time) (time.Duration, error) {
	if targetSlot == phase0.Slot(0) {
		return 0, nil
	}

	targetSlotStartTime, err := s.blockchain.SlotStartTime(targetSlot)
	if err != nil {
		return 0, fmt.Errorf("get target slot start time: %w", err)
	}

	return lineTimestamp.Sub(targetSlotStartTime), nil
}

// specialSyncCommitteeContributionDutyLines highlights certain duty-relevant log-lines that will be skipped (filtered out) by
// other rules we have defined.
func specialSyncCommitteeContributionDutyLines(line string) bool {
	// This is a special handling of legacy log-line (that contains "got duties").
	if strings.Contains(line, "got duties") && strings.Contains(line, "\"handler\":\"SYNC_COMMITTEE\"") {
		return true
	}

	// This is a special handling of legacy log-line (that contains "ticker event").
	if strings.Contains(line, "ticker event") && strings.Contains(line, "\"handler\":\"SYNC_COMMITTEE\"") {
		return true
	}

	return false
}

func relevantSyncCommitteeContributionDutyStep(line string) bool {
	// TODO - gotta filter by validator-pubkey sometimes as well ?
	//const vPubkey = "903dff3e6a2615754803e58e320d206056535c354c1b650793b0c14c00017de4fc341b25869928a83a3bcaa45f943379"
	//if !strings.Contains(line, vPubkey) {
	//	return false
	//}

	if !maybeRelevantForSyncCommitteeContribution(line) {
		return false
	}

	for _, dutyStep := range dutyStepsSyncCommitteeContribution {
		if strings.Contains(line, dutyStep) {
			return true
		}
	}

	return false
}

func maybeRelevantForSyncCommitteeContribution(line string) bool {
	return helper.ContainsCaseInsensitive(line, "sync_committee")
}
