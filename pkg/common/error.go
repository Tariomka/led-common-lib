package common

import "errors"

var (
	ErrOutOfBounds        = errors.New("index out of bounds")
	ErrNotEnoughData      = errors.New("not enough data")
	ErrUnsupportedVersion = errors.New("unsupported version")
)
