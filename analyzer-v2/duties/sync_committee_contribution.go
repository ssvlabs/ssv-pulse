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
	"broadcasted post consensus partial signature message",
	"got post consensus quorum",
	"submitting sync committee aggregator",
	"successfully submitted sync committee aggregator",
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

func (s *SyncCommitteeContribution) Analyze(logFilePath string, targetSlot phase0.Slot) error {
	logFile, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	logger := slog.With("duty_type", "syncCommitteeContribution")

	lineNumber := 0
	scanner := parser.NewScanner(logFile)
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		lineIsRelevant := func() bool {
			if !relevantForSlot(line, targetSlot) {
				return false
			}

			if containsUnexpectedError(line) || containsUnexpectedWarn(line) {
				return true
			}

			return relevantForSyncCommitteeContributionDuty(line)
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

func relevantForSyncCommitteeContributionDuty(line string) bool {
	if !helper.ContainsCaseInsensitive(line, "sync_committee") {
		return false
	}

	for _, dutyStep := range dutyStepsSyncCommitteeContribution {
		if strings.Contains(line, dutyStep) {
			return true
		}
	}

	return false
}
