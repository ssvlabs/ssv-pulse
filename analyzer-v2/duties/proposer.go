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

var dutyStepsProposer = []string{
	"ticker event",
	"got duties",
	"executing validator duty",
	"late duty execution",
	"starting duty processing",
	"signed & broadcasted partial RANDAO signature",
	"got partial RANDAO signatures",
	"got pre consensus quorum",
	"reconstructed partial RANDAO signatures",
	"got beacon block proposal",
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
	"waited out proposer delay",
	"submitting block proposal",
	"successfully submitted block proposal",
	"successfully finished duty processing",
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

func (s *Proposer) Analyze(logFilePath string, targetSlot phase0.Slot) error {
	logFile, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	logger := slog.With("duty_type", "proposer")

	lineNumber := 0
	scanner := parser.NewScanner(logFile)
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		if !relevantForSlot(line, targetSlot) {
			continue
		}
		if !relevantForProposerDuty(line) {
			continue
		}

		lineIsRelevant := false
		for _, dutyStep := range dutyStepsProposer {
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

func (s *Proposer) logWithTimeIntoSlot(logger *slog.Logger, line string, lineNumber int, targetSlot phase0.Slot) error {
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
