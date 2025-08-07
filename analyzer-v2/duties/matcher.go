package duties

import (
	"strings"

	"github.com/ssvlabs/ssv-pulse/analyzer-v2/internal/helper"
)

var dutyStepsCommittee = []string{
	"starting duty processing",
	"fetched attestation data from CL",
	"round timed out",
	"QBFT instance decided",
	"constructed & signed post consensus partial signature message",
	"broadcasted post consensus partial signature message",
	"got post consensus quorum",
	"submitting attestations",
	"successfully submitted attestations",
	"submitting sync committee",
	"successfully submitted sync committee",
	"successfully finished duty processing",
}

func containsUnexpectedCommitteeError(line string) bool {
	return containsUnexpectedError(line)
}

var dutyStepsProposer = []string{
	"starting duty processing",
	"got partial RANDAO signatures",
	"reconstructed partial RANDAO signatures",
	"got beacon block proposal",
	"round timed out",
	//"QBFT instance decided",
	//"broadcasted post consensus partial signature message",
	//"got post consensus quorum",
	//"submitting attestations",
	"reconstructed partial post consensus signatures proposer", // TODO - remove ? we don't need it
	"waited out proposer delay",
	//"submitting block proposal",
	"successfully submitted block proposal",
	//"successfully finished duty processing",
}

func containsUnexpectedProposerError(line string) bool {
	return containsUnexpectedError(line)
}

const (
	dutyTypeCommitteePattern     = "committee"
	dutyTypeProposerPattern      = "proposer"
	dutyTypeSyncCommitteePattern = "sync_committee"

	slotPattern = "\"slot\":%d"
)

func containsUnexpectedError(line string) bool {
	if !helper.ContainsCaseInsensitive(line, "err") {
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
