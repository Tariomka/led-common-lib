package uart

import (
	"io"

	"github.com/Tariomka/led-common-lib/pkg/common"
)

const (
	// ShowAddress is the flash address of show data offset 0 in upload records.
	ShowAddress = 0x0800B000
	// ShowCapacity is the size of the cube's show region.
	ShowCapacity = 160000
	// RecordSize is the most data bytes sent in one record.
	RecordSize = 128

	msgConnect = 17
	msgReady   = 16
	msgRecord  = 32
	msgStored  = 33
	msgEnd     = 37
)

type UploadState int

const (
	UploadIdle       UploadState = iota
	UploadConnecting             // connect frame sent, waiting for the ready frame
	UploadSending                // sending records as data arrives
	UploadEnding                 // end frames sent, waiting for the cube's result line
	UploadDone
	UploadFailed
)

// Upload stores show data on the cube. It is a state machine run by the caller's loop: Begin, Write
// the data in pieces of any size as it arrives, then Finish. Pass every frame and text line read
// from the cube to Frame and Line, and call Tick regularly for the timeouts. A record without a
// reply within Timeout is sent once more before the upload fails. A failed upload sends no end
// frame, so the cube doesn't load partial data.
type Upload struct {
	out      io.Writer
	state    UploadState
	err      error
	size     int
	queued   []byte // data not acknowledged yet, starting at show offset acked
	acked    int
	pending  int // bytes of queued in the record waiting for its reply, 0 = none
	line     uint16
	retried  bool
	deadline uint32
	finish   bool
	payload  []byte
	frame    []byte
}

// NewUpload creates an upload client writing frames to out.
func NewUpload(out io.Writer) *Upload {
	return &Upload{out: out}
}

func (this *Upload) State() UploadState { return this.state }

// Err is the reason of a failed upload.
func (this *Upload) Err() error { return this.err }

// Acked is the number of bytes the cube has stored.
func (this *Upload) Acked() int { return this.acked }

// Queued is the number of bytes given to Write so far, i.e. the offset of the next Write.
func (this *Upload) Queued() int { return this.acked + len(this.queued) }

// Total is the size given to Begin.
func (this *Upload) Total() int { return this.size }

func (this *Upload) active() bool {
	return this.state == UploadConnecting || this.state == UploadSending || this.state == UploadEnding
}

// Begin starts an upload of size bytes; now is a millisecond clock.
func (this *Upload) Begin(size int, now uint32) error {
	if this.active() {
		return common.ErrUploadBusy
	}
	if size < 0 || size > ShowCapacity {
		return common.ErrUploadSize
	}

	this.state, this.err, this.size = UploadConnecting, nil, size
	this.queued, this.acked, this.pending, this.line, this.retried, this.finish = this.queued[:0], 0, 0, 0, false, false
	this.connect(now)
	return this.err
}

// Write queues the next piece of show data.
func (this *Upload) Write(data []byte, now uint32) error {
	if (this.state != UploadConnecting && this.state != UploadSending) || this.finish {
		return common.ErrUploadState
	}
	if this.Queued()+len(data) > ShowCapacity {
		this.fail(common.ErrUploadSize)
		return common.ErrUploadSize
	}

	this.queued = append(this.queued, data...)
	this.next(now)
	return this.err
}

// Finish ends the upload once all queued data is stored.
func (this *Upload) Finish(now uint32) error {
	if (this.state != UploadConnecting && this.state != UploadSending) || this.finish {
		return common.ErrUploadState
	}

	this.finish = true
	this.next(now)
	return this.err
}

// Abort stops the upload without telling the cube.
func (this *Upload) Abort() {
	if this.active() {
		this.state, this.err = UploadFailed, common.ErrUploadState
	}
}

// Frame handles the payload of a frame received from the cube.
func (this *Upload) Frame(payload []byte, now uint32) {
	if len(payload) == 0 {
		return
	}

	switch {
	case this.state == UploadConnecting && payload[0] == msgReady:
		if len(payload) > 1 && payload[1] != 0 {
			this.fail(common.ErrUploadRefused)
			return
		}

		this.state = UploadSending
		this.next(now)
	case this.state == UploadSending && payload[0] == msgStored && this.pending > 0 && len(payload) >= 4 &&
		payload[1] == byte(this.pending) && uint16(payload[2])<<8|uint16(payload[3]) == this.line:
		this.queued = this.queued[:copy(this.queued, this.queued[this.pending:])]
		this.acked += this.pending
		this.pending, this.retried = 0, false
		this.line++
		this.next(now)
	}
}

// Line handles a text line received from the cube.
func (this *Upload) Line(line string) {
	if this.state != UploadEnding {
		return
	}

	switch line {
	case LineUploadOK:
		this.state = UploadDone
	case LineUploadFailed:
		this.fail(common.ErrUploadFailed)
	}
}

// Tick handles the reply timeouts.
func (this *Upload) Tick(now uint32) {
	waiting := this.state == UploadConnecting || this.state == UploadEnding || this.pending > 0
	if !this.active() || !waiting || int32(now-this.deadline) < 0 {
		return
	}

	if this.retried || this.state == UploadEnding {
		this.fail(common.ErrUploadTimeout)
		return
	}

	this.retried = true
	if this.state == UploadConnecting {
		this.connect(now)
	} else {
		this.record(now)
	}
}

func (this *Upload) fail(err error) {
	if this.active() {
		this.state, this.err = UploadFailed, err
	}
}

func (this *Upload) send(now uint32) {
	this.frame = AppendFrame(this.frame[:0], this.payload)
	if _, err := this.out.Write(this.frame); err != nil {
		this.fail(err)
	}
	this.deadline = now + Timeout
}

// connect sends [17, md5[16], size u32]; the cube ignores md5 and size.
func (this *Upload) connect(now uint32) {
	size := uint32(this.size)
	var md5 [16]byte
	this.payload = append(this.payload[:0], msgConnect)
	this.payload = append(this.payload, md5[:]...)
	this.payload = append(this.payload, byte(size>>24), byte(size>>16), byte(size>>8), byte(size))
	this.send(now)
}

// record sends [32, n, line u16 BE, address u32 BE, data] for the pending record.
func (this *Upload) record(now uint32) {
	address := uint32(ShowAddress + this.acked)
	this.payload = append(this.payload[:0], msgRecord, byte(this.pending), byte(this.line>>8), byte(this.line),
		byte(address>>24), byte(address>>16), byte(address>>8), byte(address))
	this.payload = append(this.payload, this.queued[:this.pending]...)
	this.send(now)
}

func (this *Upload) end(now uint32) {
	this.payload = append(this.payload[:0], msgEnd, 0)
	this.send(now)
}

// next sends the next record, or the end frames when everything is stored.
func (this *Upload) next(now uint32) {
	if this.state != UploadSending || this.pending > 0 {
		return
	}

	if len(this.queued) > 0 {
		this.pending, this.retried = min(len(this.queued), RecordSize), false
		this.record(now)
		return
	}

	if this.finish {
		this.end(now)
		this.end(now)
		if this.state == UploadSending {
			this.state = UploadEnding
		}
	}
}
