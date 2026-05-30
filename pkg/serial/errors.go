package serial

import "errors"

var (
	ErrPortNotOpen = errors.New("serial port not open")
)