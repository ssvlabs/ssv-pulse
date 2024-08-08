package configs

import "errors"

type Address string

func (a Address) Validate() error {
	if a == "" {
		return errors.New("address is an empty string")
	}
	return nil
}
