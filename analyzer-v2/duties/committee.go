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

var dutyStepsCommittee = []string{
	"got duties",
	"executing committee duty",
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

func (s *Committee) Analyze(logFilePath string, targetSlot phase0.Slot) error {
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

		if !relevantForSlot(line, targetSlot) {
			continue
		}
		if !relevantForCommitteeDuty(line) {
			continue
		}

		lineIsRelevant := false
		for _, dutyStep := range dutyStepsCommittee {
			if strings.Contains(line, dutyStep) {
				lineIsRelevant = true
				break
			}
		}
		if lineIsRelevant {
			if err := s.logWithTimeIntoSlot(logger, line, lineNumber, targetSlot); err != nil {
				return err
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

	entry, trimmedLine, err := s.logParser.ParseLogLine(line)
	if err != nil {
		return fmt.Errorf("parse log line %d `%s`, err: %w", lineNumber, line, err)
	}
	timeIntoSlot := entry.Timestamp.Sub(targetSlotStartTime)

	timeIntoSlotStr := fmt.Sprintf("time_into_slot_ms=%d", timeIntoSlot.Milliseconds())
	logger.Info(timeIntoSlotStr + " " + trimmedLine)

	return nil
}
