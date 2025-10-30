package helper

import (
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

var (
	slotRegexpPrefix1 = `"slot":`
	slotRegexpSuffix1 = `,"`
	slotRegexpPrefix2 = `-s`
	slotRegexpSuffix2 = `",`
	slotRegexp1       = regexp.MustCompile(fmt.Sprintf(`%s[^,]*%s`, slotRegexpPrefix1, slotRegexpSuffix1))
	slotRegexp2       = regexp.MustCompile(fmt.Sprintf(`%s[^,]*%s`, slotRegexpPrefix2, slotRegexpSuffix2))
)

func TryParseSlot(line string) phase0.Slot {
	if m := slotRegexp1.FindStringSubmatch(line); len(m) >= 1 {
		slotStr := m[0][len(slotRegexpPrefix1) : len(m[0])-len(slotRegexpSuffix1)]
		slot, err := strconv.ParseUint(slotStr, 10, 64)
		if err != nil {
			return phase0.Slot(math.MaxUint64) // to gently signal the parsing issue
		}
		return phase0.Slot(slot)
	}
	if m := slotRegexp2.FindStringSubmatch(line); len(m) >= 1 {
		slotStr := m[0][len(slotRegexpPrefix2) : len(m[0])-len(slotRegexpSuffix2)]
		slot, err := strconv.ParseUint(slotStr, 10, 64)
		if err != nil {
			return phase0.Slot(math.MaxUint64) // to gently signal the parsing issue
		}
		return phase0.Slot(slot)
	}
	return phase0.Slot(0)
}
