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

var dutyStepsAggregator = []string{
	"ticker event",
	"got duties",
	"executing validator duty",
	"late duty execution",
	"starting duty processing",
	"signed aggregator selection proof",
	"got pre consensus quorum",
	"got partial aggregator selection proof signatures",
	"aggregation duty won't be needed from this validator for this slot",
	"submitted aggregate and proof",
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
	"submitting signed aggregate and proof",
	"successful submitted aggregate", // TODO - remove this line once typo-fix is enacted (`successful` -> `successfully`)
	"successfully submitted signed aggregate and proof",
	"successfully finished duty processing",
}

type Aggregator struct {
	blockchain *environment.Blockchain
	logParser  environment.LogParser
}

func NewAggregator(blockchain *environment.Blockchain, logParser environment.LogParser) *Aggregator {
	return &Aggregator{
		blockchain: blockchain,
		logParser:  logParser,
	}
}

func (s *Aggregator) Analyze(logFilePath string, dutyID string, targetSlot phase0.Slot) error {
	logFile, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	logger := slog.With("duty_type", "aggregator")

	lineNumber := 0
	scanner := parser.NewScanner(logFile)
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		lineIsRelevant := func() bool {
			if !relevantForSlot(line, targetSlot) {
				return false
			}

			if containsUnexpectedAggregatorError(line) {
				return true
			}

			return relevantForAggregatorDuty(line)
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

func (s *Aggregator) logWithTimeIntoSlot(logger *slog.Logger, line string, lineNumber int, targetSlot phase0.Slot) error {
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

func relevantForAggregatorDuty(line string) bool {
	// TODO - gotta filter by validator-pubkey sometimes as well ?
	//const vPubkey = "903dff3e6a2615754803e58e320d206056535c354c1b650793b0c14c00017de4fc341b25869928a83a3bcaa45f943379"
	//if !strings.Contains(line, vPubkey) {
	//	return false
	//}

	if !maybeRelevantForAggregator(line) {
		return false
	}

	for _, dutyStep := range dutyStepsAggregator {
		if strings.Contains(line, dutyStep) {
			return true
		}
	}

	return false
}

func maybeRelevantForAggregator(line string) bool {
	return helper.ContainsCaseInsensitive(line, "aggregator")
}
