package parser

import "time"

func IsCluster(operators []SignerID, signers map[SignerID]time.Time) bool {
	if len(operators) < 4 || len(operators) > 13 || len(operators)%2 != 1 {
		return false
	}

	if len(operators) != len(signers) {
		return false
	}

	for _, id := range operators {
		_, exist := signers[id]
		if !exist {
			return false
		}
	}
	return true
}
