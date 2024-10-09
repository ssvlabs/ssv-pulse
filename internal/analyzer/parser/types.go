package parser

import (
	"errors"
	"time"
)

type (
	SignerID        = uint32
	DutyID          = string
	MultiFormatTime struct {
		time.Time
	}
)

var customTimeLayouts = []string{
	"2006-01-02T15:04:05.000-0700",
	"2006-01-02T15:04:05.000Z",
}

func (m *MultiFormatTime) UnmarshalJSON(b []byte) error {
	str := string(b)
	str = str[1 : len(str)-1]

	var parseErr error
	for _, layout := range customTimeLayouts {
		t, err := time.Parse(layout, str)
		if err == nil {
			m.Time = t
			return nil
		}
		parseErr = err
	}

	return errors.Join(parseErr, errors.New("unable to parse time"))
}
