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

var dutyStepsSyncCommitteeContribution = []string{
	"ticker event",
	"got duties",
	"executing validator duty",
	"late duty execution",
	"starting duty processing",
	"got pre consensus quorum",
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
	"broadcasted post consensus partial signature message",
	"got post consensus quorum",
	"submitting sync committee contributions",
	"successfully submitted sync committee contributions",
	"successfully finished duty processing",
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

	logger := slog.With("duty_type", "sync_committee_contribution")

	lineNumber := 0
	scanner := parser.NewScanner(logFile)
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		lineIsRelevant := func() bool {
			if containsUnexpectedSyncCommitteeContributionError(line) && (relevantForDutyID(line, dutyID) || relevantForSlot(line, targetSlot)) {
				return true
			}

			// Special lines are relevant only if the target slot has been specified.
			if specialSyncCommitteeContributionDutyLines(line) && relevantForSlot(line, targetSlot) {
				return true
			}

			// The line is interesting only if it references a specific duty-step, the rest would be noise.
			if relevantSyncCommitteeContributionDutyStep(line) && (dutyID != "" && relevantForDutyID(line, dutyID)) {
				return true
			}

			// The line is interesting only if it references a specific duty-step, the rest would be noise.
			if relevantSyncCommitteeContributionDutyStep(line) && relevantForSlot(line, targetSlot) {
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

func (s *SyncCommitteeContribution) logWithTimeIntoSlot(logger *slog.Logger, line string, lineNumber int, targetSlot phase0.Slot) error {
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

func relevantForSyncCommitteeContribution(line string) bool {
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
