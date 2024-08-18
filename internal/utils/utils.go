package utils

import (
	"fmt"
	"sort"
	"strconv"
)

// StingSliceToUintArray converts the string slice to uint64 slice
func StingSliceToUintArray(flagdata []string) ([]uint64, error) {
	partsarr := make([]uint64, 0, len(flagdata))
	for i := 0; i < len(flagdata); i++ {
		opid, err := strconv.ParseUint(flagdata[i], 10, strconv.IntSize)
		if err != nil {
			return nil, fmt.Errorf("😥 cant parse uint from string: %v , data: %v, ", err, flagdata[i])
		}
		partsarr = append(partsarr, opid)
	}
	// sort array
	sort.SliceStable(partsarr, func(i, j int) bool {
		return partsarr[i] < partsarr[j]
	})
	sorted := sort.SliceIsSorted(partsarr, func(p, q int) bool {
		return partsarr[p] < partsarr[q]
	})
	if !sorted {
		return nil, fmt.Errorf("slice isnt sorted")
	}
	return partsarr, nil
}
