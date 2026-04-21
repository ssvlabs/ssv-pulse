package helper

import (
	"fmt"
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

func TryParseSlot(line string) (phase0.Slot, error) {
	if m := slotRegexp1.FindStringSubmatch(line); len(m) >= 1 {
		slotStr := m[0][len(slotRegexpPrefix1) : len(m[0])-len(slotRegexpSuffix1)]
		slot, err := strconv.ParseUint(slotStr, 10, 64)
		if err != nil {
			return phase0.Slot(0), err
		}
		return phase0.Slot(slot), nil
	}
	if m := slotRegexp2.FindStringSubmatch(line); len(m) >= 1 {
		slotStr := m[0][len(slotRegexpPrefix2) : len(m[0])-len(slotRegexpSuffix2)]
		slot, err := strconv.ParseUint(slotStr, 10, 64)
		if err != nil {
			return phase0.Slot(0), err
		}
		return phase0.Slot(slot), nil
	}
	return phase0.Slot(0), nil
}
