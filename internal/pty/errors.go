package pty

import "errors"

var (
	ErrPTYNotOpen = errors.New("PTY not open")
)