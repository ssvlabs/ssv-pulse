package parser

import (
	"errors"
	"time"
)

type (
	//Signer ID is the same as Operator ID
	SignerID = uint32

	/*
		ATTESTER-e79985-s2559525-v1664056
		AGGREGATOR-e82122-s2627920-v1805391
		VALIDATOR_REGISTRATION-e82098-s2627136-v1805376
		e - epoch
		s - slot
		v - validator index
	*/
	DutyID = string

	MultiFormatTime struct {
		time.Time
	}

	Metric[T any] struct {
		Found bool
		Value T
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
