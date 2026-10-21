package tcp

import (
	"encoding/binary"

	"github.com/Tariomka/led-common-lib/pkg/common"
)

// MaxChunk is the most show data bytes in one UploadChunk message.
const MaxChunk = 2048

func newMessage(messageType Type, payload []byte) Message {
	return Message{Version: Version, Type: messageType, Payload: payload}
}

func NewHello(name string) Message { return newMessage(TypeHello, []byte(name)) }

func NewCommand(line string) Message { return newMessage(TypeCommand, []byte(line)) }

func NewCubeLine(line string) Message { return newMessage(TypeCubeLine, []byte(line)) }

func NewPicoLine(line string) Message { return newMessage(TypePicoLine, []byte(line)) }

func NewUploadBegin(size uint32) Message {
	return newMessage(TypeUploadBegin, binary.BigEndian.AppendUint32(nil, size))
}

func NewUploadChunk(offset uint32, data []byte) Message {
	payload := binary.BigEndian.AppendUint32(make([]byte, 0, 4+len(data)), offset)
	return newMessage(TypeUploadChunk, append(payload, data...))
}

func NewUploadEnd() Message { return newMessage(TypeUploadEnd, nil) }

func NewUploadProgress(acked, total uint32) Message {
	return newMessage(TypeUploadProgress, binary.BigEndian.AppendUint32(binary.BigEndian.AppendUint32(nil, acked), total))
}

func NewUploadResult(ok bool, message string) Message {
	payload := []byte{0}
	if ok {
		payload[0] = 1
	}
	return newMessage(TypeUploadResult, append(payload, message...))
}

func NewPing() Message { return newMessage(TypePing, nil) }

func NewPong() Message { return newMessage(TypePong, nil) }

// Text is the payload of Hello, Command, CubeLine and PicoLine messages.
func (this Message) Text() string { return string(this.Payload) }

// Size reads an UploadBegin payload.
func (this Message) Size() (uint32, error) {
	if len(this.Payload) < 4 {
		return 0, common.ErrNotEnoughData
	}

	return binary.BigEndian.Uint32(this.Payload), nil
}

// Chunk reads an UploadChunk payload; data shares the payload's memory.
func (this Message) Chunk() (offset uint32, data []byte, err error) {
	if len(this.Payload) < 4 {
		return 0, nil, common.ErrNotEnoughData
	}

	return binary.BigEndian.Uint32(this.Payload), this.Payload[4:], nil
}

// Progress reads an UploadProgress payload.
func (this Message) Progress() (acked, total uint32, err error) {
	if len(this.Payload) < 8 {
		return 0, 0, common.ErrNotEnoughData
	}

	return binary.BigEndian.Uint32(this.Payload), binary.BigEndian.Uint32(this.Payload[4:]), nil
}

// Result reads an UploadResult payload.
func (this Message) Result() (ok bool, message string, err error) {
	if len(this.Payload) < 1 {
		return false, "", common.ErrNotEnoughData
	}

	return this.Payload[0] != 0, string(this.Payload[1:]), nil
}
