package duties

import (
	"strings"

	"github.com/ssvlabs/ssv-pulse/analyzer-v2/internal/helper"
)

var dutyStepsCommittee = []string{
	"starting duty processing",
	"fetched attestation data from CL",
	"starting new QBFT instance",
	"round timed out",
	"got justified round change",
	"got commit quorum",
	"QBFT instance is decided",
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
	"signed & broadcasted partial RANDAO signature",
	"got partial RANDAO signatures",
	"reconstructed partial RANDAO signatures",
	"got beacon block proposal",
	"starting new QBFT instance",
	"round timed out",
	"got justified round change",
	"got commit quorum",
	"QBFT instance is decided",
	"broadcasted post consensus partial signature message",
	"got post consensus quorum",
	"waited out proposer delay",
	"submitting block proposal",
	"successfully submitted block proposal",
	"successfully finished duty processing",
}

func containsUnexpectedProposerError(line string) bool {
	return containsUnexpectedError(line)
}

var dutyStepsAggregator = []string{
	"starting duty processing",
	"signed aggregator selection proof",
	"got partial aggregator selection proof signatures",
	"aggregation duty won't be needed from this validator for this slot",
	"submitted aggregate and proof",
	"starting new QBFT instance",
	"round timed out",
	"got justified round change",
	"got commit quorum",
	"QBFT instance is decided",
	"broadcasted post consensus partial signature message",
	"got post consensus quorum",
	"submitting signed aggregate and proof",
	"successful submitted aggregate", // TODO - remove this line once typo-fix is enacted (`successful` -> `successfully`)
	"successfully submitted signed aggregate and proof",
	"successfully finished duty processing",
}

func containsUnexpectedAggregatorError(line string) bool {
	return containsUnexpectedError(line)
}

const (
	dutyTypeCommitteePattern     = "committee"
	dutyTypeProposerPattern      = "proposer"
	dutyTypeAggregatorPattern    = "aggregator"
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
