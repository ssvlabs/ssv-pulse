package duties

import (
	"fmt"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ssvlabs/ssv-pulse/analyzer-v2/internal/helper"
)

func relevantForCommitteeDuty(line string) bool {
	// Clean up the line from false-positive triggers it potentially might have.
	line = strings.ReplaceAll(line, "\"committee_index\":", "")
	line = strings.ReplaceAll(line, "\"handler\":\"SYNC_COMMITTEE\"", "")

	// This is a special handling of legacy log-line (that contains "ticker event").
	if strings.Contains(line, "\"handler\":\"CLUSTER\"") {
		return true
	}
	// This is a special handling of legacy log-line (that contains "got duties").
	if strings.Contains(line, "\"handler\":\"ATTESTER\"") && strings.Contains(line, "\"duties\":\"") {
		return true
	}

	if containsUnexpectedError(line) || containsUnexpectedWarn(line) {
		return true
	}
	return helper.ContainsCaseInsensitive(line, "committee")
}

func relevantForProposerDuty(line string) bool {
	// TODO - gotta filter by validator-pubkey sometimes as well ?
	//const vPubkey = "903dff3e6a2615754803e58e320d206056535c354c1b650793b0c14c00017de4fc341b25869928a83a3bcaa45f943379"
	//if !strings.Contains(line, vPubkey) {
	//	return false
	//}
	if containsUnexpectedError(line) || containsUnexpectedWarn(line) {
		return true
	}
	return helper.ContainsCaseInsensitive(line, "proposer")
}

func relevantForAggregatorDuty(line string) bool {
	// TODO - gotta filter by validator-pubkey sometimes as well ?
	//const vPubkey = "903dff3e6a2615754803e58e320d206056535c354c1b650793b0c14c00017de4fc341b25869928a83a3bcaa45f943379"
	//if !strings.Contains(line, vPubkey) {
	//	return false
	//}
	if containsUnexpectedError(line) || containsUnexpectedWarn(line) {
		return true
	}
	return helper.ContainsCaseInsensitive(line, "aggregator")
}

func relevantForSyncCommitteeContributionDuty(line string) bool {
	if containsUnexpectedError(line) || containsUnexpectedWarn(line) {
		return true
	}
	return helper.ContainsCaseInsensitive(line, "sync_committee")
}

func relevantForSlot(line string, targetSlot phase0.Slot) bool {
	if strings.Contains(line, fmt.Sprintf("\"slot\":%d", targetSlot)) {
		return true
	}
	if strings.Contains(line, fmt.Sprintf("-s%d", targetSlot)) {
		return true
	}
	return false
}

func containsUnexpectedError(line string) bool {
	if !helper.ContainsCaseInsensitive(line, "err") {
		return false
	}

	// TODO - this error should no longer show up with the latest changes (merged into stage branch,
	// but not into main branch yet), so we'll need to remove the skipping of this error eventually.
	if strings.Contains(line, "validator is not an aggregator") {
		return false
	}

	if strings.Contains(line, "consensus has already finished") {
		return false
	}

	if strings.Contains(line, "invalid post-consensus message: no running duty") {
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

func containsUnexpectedWarn(line string) bool {
	if !helper.ContainsCaseInsensitive(line, "warn") {
		return false
	}

	return true
}
