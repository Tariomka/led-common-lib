package common

import "errors"

// Desktop <-> RPi Pico protocol (communication/tcp)
var (
	ErrNotEnoughData      = errors.New("not enough data")
	ErrUnsupportedVersion = errors.New("unsupported version")
	ErrPayloadTooLarge    = errors.New("tcp: payload too large")
)

// Show data (lightshow/blob)
var (
	ErrBlobMagic       = errors.New("blob: bad magic")
	ErrBlobLayout      = errors.New("blob: section outside data")
	ErrBlobCompression = errors.New("blob: bad graph data")
	ErrBlobNameTooLong = errors.New("blob: name longer than 15 bytes")
	ErrBlobTooLarge    = errors.New("blob: show data larger than 160000 bytes")
	ErrBlobValue       = errors.New("blob: value out of range")
)

// Cube protocol (communication/uart)
var (
	// ErrCommand reports a command line the cube can't receive: empty, too long or containing '?'
	// (the cube ends a command at "?!" and drops it on any other '?').
	ErrCommand       = errors.New("uart: invalid command line")
	ErrUploadBusy    = errors.New("uart: upload already running")
	ErrUploadState   = errors.New("uart: upload not accepting data")
	ErrUploadSize    = errors.New("uart: show data larger than the cube's show region")
	ErrUploadTimeout = errors.New("uart: cube did not answer")
	ErrUploadRefused = errors.New("uart: cube refused the upload")
	ErrUploadFailed  = errors.New("uart: cube reported a failed upload")
)
