package duties

import (
	"fmt"
	"strings"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ssvlabs/ssv-pulse/analyzer-v2/internal/helper"
)

func relevantForSlot(line string, targetSlot phase0.Slot) bool {
	// The line is not relevant if the target slot isn't specified.
	if targetSlot == phase0.Slot(0) {
		return false
	}

	// See if the line is relevant to that slot number.
	if strings.Contains(line, fmt.Sprintf("\"slot\":%d", targetSlot)) {
		return true
	}
	if strings.Contains(line, fmt.Sprintf("-s%d", targetSlot)) {
		return true
	}

	return false
}

func maybeRelevantForSlot(line string, targetSlot phase0.Slot, timeIntoSlot time.Duration) bool {
	if relevantForSlot(line, targetSlot) {
		return true
	}

	// See if the line is relevant timewise (we are interested in the current slot plus small buffer before/after it).
	return -4*time.Second < timeIntoSlot && timeIntoSlot < 12*time.Second+4*time.Second
}

func containsUnexpectedAggregatorError(line string) bool {
	if !containsUnexpectedError(line) {
		return false
	}

	// We could try to filter by `aggregator` here - but that would skip over many potentially
	// relevant errors ... instead we'll filter out know false-positives.
	if maybeRelevantForCommittee(line) {
		return false
	}
	if maybeRelevantForProposer(line) {
		return false
	}
	if maybeRelevantForSyncCommitteeContribution(line) {
		return false
	}

	return true
}

func containsUnexpectedCommitteeError(line string) bool {
	if !containsUnexpectedError(line) {
		return false
	}

	// We could try to filter by `aggregator` here - but that would skip over many potentially
	// relevant errors ... instead we'll filter out know false-positives.
	if maybeRelevantForAggregator(line) {
		return false
	}
	if maybeRelevantForProposer(line) {
		return false
	}
	if maybeRelevantForSyncCommitteeContribution(line) {
		return false
	}

	return true
}

func containsUnexpectedProposerError(line string) bool {
	if !containsUnexpectedError(line) {
		return false
	}

	// We could try to filter by `aggregator` here - but that would skip over many potentially
	// relevant errors ... instead we'll filter out know false-positives.
	if maybeRelevantForAggregator(line) {
		return false
	}
	if maybeRelevantForCommittee(line) {
		return false
	}
	if maybeRelevantForSyncCommitteeContribution(line) {
		return false
	}

	return true
}

func containsUnexpectedSyncCommitteeContributionError(line string) bool {
	if !containsUnexpectedError(line) {
		return false
	}

	// We could try to filter by `aggregator` here - but that would skip over many potentially
	// relevant errors ... instead we'll filter out know false-positives.
	if maybeRelevantForAggregator(line) {
		return false
	}
	if maybeRelevantForCommittee(line) {
		return false
	}
	if maybeRelevantForProposer(line) {
		return false
	}

	return true
}

func containsUnexpectedError(line string) bool {
	// Clean up the line from false-positive triggers it potentially might have.
	line = strings.ReplaceAll(line, "\"errored\":0", "")

	// Treat warnings as errors.
	if !helper.ContainsCaseInsensitive(line, "err") && !helper.ContainsCaseInsensitive(line, "warn") {
		return false
	}

	// TODO - what is that ?
	if strings.Contains(line, "validator registration") {
		return false
	}

	if strings.Contains(line, "retrying message") {
		return false
	}

	if strings.Contains(line, "consensus has already finished") {
		return false
	}

	if strings.Contains(line, "invalid partial sig slot") {
		return false
	}

	if strings.Contains(line, "invalid post-consensus message: no running duty") {
		return false
	}

	if strings.Contains(line, "invalid post-consensus message: no decided value") {
		return false
	}

	if strings.Contains(line, "not processing consensus message since instance is already decided") {
		return false
	}

	if strings.Contains(line, "instance stopped processing messages") {
		return false
	}

	return true
}

func relevantForDutyID(line string, dutyID string) bool {
	// The line is not relevant if the duty ID isn't specified.
	if dutyID == "" {
		return false
	}
	return helper.ContainsCaseInsensitive(line, dutyID)
}
