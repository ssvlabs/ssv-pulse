package ssv

func IsValidClusterSize(operators []uint32) bool {
	if len(operators) < 4 || len(operators) > 13 || len(operators)%3 != 1 {
		return false
	}

	return true
}
