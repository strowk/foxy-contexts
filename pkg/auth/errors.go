package auth

import "errors"

var (
	ErrInvalidTransport = errors.New("invalid transport specified, only remote transports have authorization support")
)
