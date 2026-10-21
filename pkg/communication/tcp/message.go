// Package tcp is the protocol between the desktop app and the Pico W bridge. A message is
// [version u8][type u8][length u16 BE][payload]. The header layout never changes, so a peer can
// always read the version and skip or handle messages of another protocol version.
package tcp

import (
	"io"

	"github.com/Tariomka/led-common-lib/pkg/common"
)

// Version is the protocol version this package writes.
const Version = 1

const (
	HeaderSize = 4
	MaxPayload = 0xFFFF
)

type Type byte

const (
	TypeHello          Type = 0x01 // both ways: peer name
	TypeCommand        Type = 0x02 // desktop -> Pico: console command line, sent to the cube as "#?line?!"
	TypeCubeLine       Type = 0x03 // Pico -> desktop: a text line received from the cube
	TypePicoLine       Type = 0x04 // Pico -> desktop: a log line of the Pico
	TypeUploadBegin    Type = 0x10 // desktop -> Pico: total size u32
	TypeUploadChunk    Type = 0x11 // desktop -> Pico: offset u32, data (at most MaxChunk bytes)
	TypeUploadEnd      Type = 0x12 // desktop -> Pico: all chunks sent
	TypeUploadProgress Type = 0x13 // Pico -> desktop: bytes stored by the cube u32, total u32
	TypeUploadResult   Type = 0x14 // Pico -> desktop: ok u8, message
	TypePing           Type = 0x20 // both ways
	TypePong           Type = 0x21 // both ways, answer to Ping
)

type Message struct {
	Version byte
	Type    Type
	Payload []byte
}

// Marshal encodes the message with its header.
func (this Message) Marshal() ([]byte, error) {
	if len(this.Payload) > MaxPayload {
		return nil, common.ErrPayloadTooLarge
	}

	data := make([]byte, 0, HeaderSize+len(this.Payload))
	data = append(data, this.Version, byte(this.Type), byte(len(this.Payload)>>8), byte(len(this.Payload)))
	return append(data, this.Payload...), nil
}

// Write sends one message.
func Write(writer io.Writer, message Message) error {
	data, err := message.Marshal()
	if err != nil {
		return err
	}

	_, err = writer.Write(data)
	return err
}

// Read receives one message; its payload is stored in buffer and valid until buffer is reused.
// A payload longer than buffer is skipped and reported with common.ErrPayloadTooLarge. A message of
// another protocol version is returned with common.ErrUnsupportedVersion; the stream stays in
// sync either way.
func Read(reader io.Reader, buffer []byte) (Message, error) {
	var header [HeaderSize]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return Message{}, err
	}

	message := Message{Version: header[0], Type: Type(header[1])}
	length := int(header[2])<<8 | int(header[3])
	if length > len(buffer) {
		if _, err := io.CopyN(io.Discard, reader, int64(length)); err != nil {
			return message, err
		}
		return message, common.ErrPayloadTooLarge
	}

	message.Payload = buffer[:length]
	if _, err := io.ReadFull(reader, message.Payload); err != nil {
		return message, err
	}
	if message.Version != Version {
		return message, common.ErrUnsupportedVersion
	}
	return message, nil
}
