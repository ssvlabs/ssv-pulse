package analyzer

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	attestationDelay = time.Second
)

func fetchAttestationTime(attestationTimeLog string) (isDelayed bool, duration time.Duration, err error) {
	var attestationDuration time.Duration

	if strings.Contains(attestationTimeLog, "ms") {
		attestationTimeMS, e := strconv.ParseFloat(strings.Replace(attestationTimeLog, "ms", "", 2), 64)
		if e != nil {
			err = errors.Join(err, errors.New("error fetching attestation time from the log message"))
			return
		}
		attestationDuration = time.Duration(attestationTimeMS * float64(time.Millisecond))
	} else if strings.Contains(attestationTimeLog, "µs") {
		attestationTimeNS, e := strconv.ParseFloat(strings.Replace(attestationTimeLog, "µs", "", 2), 64)
		if e != nil {
			err = errors.Join(err, errors.New("error fetching attestation time from the log message"))
			return
		}
		attestationDuration = time.Duration(attestationTimeNS)
	}

	if attestationDuration != 0 {
		duration = attestationDuration

		if attestationDuration > attestationDelay {
			isDelayed = true
		}
	}

	return
}
