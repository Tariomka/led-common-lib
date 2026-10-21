// Package uart speaks the cube's UART protocol: console commands "#?line?!", the log lines the
// cube prints and the "A5 05 len payload check 55" frames of show uploads. It does no I/O of its
// own: feed it received bytes and write the bytes it returns or writes to the port.
package uart

const (
	FrameStart = 0xA5
	frameKind  = 0x05
	FrameEnd   = 0x55
	MaxPayload = 179

	// Timeout is the gap in ms after which the cube drops a partial frame or command; the reader and
	// the upload client use it too.
	Timeout = 1000
)

// AppendFrame appends the frame A5 05 len payload check 55 to dst; payload must be 1..MaxPayload bytes.
func AppendFrame(dst, payload []byte) []byte {
	check := byte(FrameStart ^ frameKind ^ FrameEnd ^ byte(len(payload)))
	for _, b := range payload {
		check ^= b
	}

	dst = append(dst, FrameStart, frameKind, byte(len(payload)))
	dst = append(dst, payload...)
	return append(dst, check, FrameEnd)
}
