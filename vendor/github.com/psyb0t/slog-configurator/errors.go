package slogconfigurator

import "errors"

var (
	ErrInvalidLogLevel  = errors.New("invalid log level")
	ErrInvalidLogFormat = errors.New("invalid log format")
)
