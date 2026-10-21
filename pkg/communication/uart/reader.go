package uart

// Item is what Reader.Feed completed.
type Item int

const (
	None  Item = iota
	Line       // a text line, without "\r\n"
	Frame      // the payload of a valid frame
)

// MaxLine is the longest text line kept; longer lines are cut.
const MaxLine = 256

const (
	readText = iota
	readMagic
	readLength
	readData
	readCheck
	readEnd
)

// Reader splits the bytes received from the cube into text lines and frames. Text and frames may
// interleave; a frame with a bad check, or with a gap longer than Timeout inside it, is dropped.
type Reader struct {
	state    int
	line     []byte
	lineDone bool
	frame    []byte
	left     int
	check    byte
	received byte
	last     uint32
	started  bool
}

// Feed processes one received byte; now is a millisecond clock. The returned data is valid until
// the next call.
func (this *Reader) Feed(b byte, now uint32) (Item, []byte) {
	if this.lineDone {
		this.line, this.lineDone = this.line[:0], false
	}
	if this.started && this.state != readText && now-this.last > Timeout {
		this.state = readText
	}
	this.started, this.last = true, now

	switch this.state {
	case readMagic:
		this.state = readText
		if b == frameKind {
			this.state, this.check = readLength, FrameStart^frameKind
			return None, nil
		}
	case readLength:
		if b == 0 || b > MaxPayload {
			this.state = readText
			return None, nil
		}
		this.state, this.left, this.check, this.frame = readData, int(b), this.check^b, this.frame[:0]
		return None, nil
	case readData:
		this.frame = append(this.frame, b)
		this.check ^= b
		if this.left--; this.left == 0 {
			this.state = readCheck
		}
		return None, nil
	case readCheck:
		this.state, this.received = readEnd, b
		return None, nil
	case readEnd:
		this.state = readText
		if b == FrameEnd && this.received == this.check^FrameEnd {
			return Frame, this.frame
		}
		return None, nil
	}

	switch b {
	case FrameStart:
		this.state = readMagic
	case '\n':
		this.lineDone = true
		line := this.line
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		return Line, line
	default:
		if len(this.line) < MaxLine {
			this.line = append(this.line, b)
		}
	}
	return None, nil
}
