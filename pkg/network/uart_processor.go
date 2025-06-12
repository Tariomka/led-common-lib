package network

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"slices"
)

var (
	ErrHandshake     = errors.New("handshake failed")
	ErrExpectedStart = errors.New("expected start marker")
	ErrExpectedEnd   = errors.New("expected end marker")
	ErrTooMuchData   = errors.New("too much data")
)

type UartDataType byte

const (
	UartEmpty UartDataType = iota
	UartMessage
	UartBytes
	UartSetting
	UartPing
	UartPong
)

var handshake = []byte("SGFuZHNoYWtl") // base64 encoded "Handshake"
var handshakeLength = len(handshake)
var ellipsis = []byte("...")

const (
	startMarker = byte('(')
	endMarker   = byte(')')

	maxRetries = 3

	bufferSize     = 1024
	nonContentSize = 1 + 1 + 2 + 1 // start marker + type + content length + end marker
	maxPayloadSize = bufferSize - nonContentSize
)

type UartProcessor struct {
	buffer  []byte // TODO: think about a [bufferSize]byte array instead of slice, to not allocate
	uart    io.ReadWriter
	synched bool
}

func NewUartProcessor(uart io.ReadWriter) *UartProcessor {
	return &UartProcessor{
		buffer: make([]byte, bufferSize),
		uart:   uart,
	}
}

func (this *UartProcessor) Read() (UartDataType, []byte, error) {
	if err := this.synchronize(); err != nil {
		return UartEmpty, nil, err
	}

	n, err := this.uart.Read(this.buffer)
	if err != nil {
		return UartEmpty, nil, err
	}

	if n == 0 {
		return UartEmpty, nil, nil
	}

	if this.buffer[0] != startMarker {
		return UartEmpty, nil, ErrExpectedStart
	}

	// TODO: keep reading?
	dType := UartDataType(this.buffer[1])
	payloadSize := binary.BigEndian.Uint16(this.buffer[2:4])
	if payloadSize > maxPayloadSize {
		return UartEmpty, nil, ErrTooMuchData
	}

	if this.buffer[payloadSize+nonContentSize-1] != endMarker {
		return UartEmpty, nil, ErrExpectedEnd
	}

	return dType, this.buffer[nonContentSize-1 : payloadSize+nonContentSize-1], nil
}

func (this *UartProcessor) WriteMessage(message string) error {
	if err := this.synchronize(); err != nil {
		return err
	}

	// TODO: desync the connection?
	return this.write(UartMessage, []byte(message))
}

func (this *UartProcessor) WriteBytes(data []byte) error {
	if err := this.synchronize(); err != nil {
		return err
	}

	// TODO: desync the connection?
	return this.write(UartBytes, data)
}

func (this *UartProcessor) SendPing() {
	this.write(UartPing, nil)
}

func (this *UartProcessor) SendPong() {
	this.write(UartPong, nil)
}

func (this *UartProcessor) write(dType UartDataType, payload []byte) error {
	var buffer bytes.Buffer
	lengthBytes := [2]byte{}
	buffer.WriteByte(startMarker)
	buffer.WriteByte(byte(dType))
	payloadLength := len(payload)
	if maxPayloadSize >= payloadLength {
		// TODO: make it pretier, maybe dry
		binary.BigEndian.PutUint16(lengthBytes[:], uint16(payloadLength))
		buffer.Write(lengthBytes[:])
		buffer.Write(payload)
		buffer.WriteByte(endMarker)
		_, err := this.uart.Write(buffer.Bytes())
		return err
	}

	binary.BigEndian.PutUint16(lengthBytes[:], maxPayloadSize-3)
	buffer.Write(lengthBytes[:])
	buffer.Write(payload[:maxPayloadSize-3])
	buffer.Write(ellipsis)
	buffer.WriteByte(endMarker)
	if _, err := this.uart.Write(buffer.Bytes()); err != nil {
		return err
	}

	return this.write(dType, append(ellipsis, payload[maxPayloadSize-3:]...))
}

func (this *UartProcessor) synchronize() error {
	if this.synched {
		return nil
	}

	if this.synched = this.establishHandshake(); !this.synched {
		return ErrHandshake
	}

	return nil
}

func (this *UartProcessor) establishHandshake() bool {
	if n, err := this.uart.Write(handshake); err != nil || n != handshakeLength {
		return false
	}

	// TODO: retry few times
	readLength := 0
	retries := 0
	for readLength < handshakeLength || retries < maxRetries {
		n, err := this.uart.Read(this.buffer)
		readLength += n
		if err != nil {
			return false
		}

		retries++
	}

	if slices.Compare(this.buffer[:handshakeLength], handshake) != 0 {
		return false
	}

	this.clearBuffer(handshakeLength)
	return true
}

// If index is outside of buffer, clears entire buffer,
// otherwise clears from start to the index, shifting the rest of the data to the start
func (this *UartProcessor) clearBuffer(index int) {
	if index < 0 || index > bufferSize {
		clear(this.buffer)
		return
	}

	clear(this.buffer[:index])
	this.buffer = append(this.buffer[index:], this.buffer[:index]...)[:bufferSize:bufferSize]
}

// TODO: add a function for formatting strings to messages for
// easy sending with print(createMessage("message"))
// probably should be something like:
// func CreateUartMessage(message string) string
// and most likely will need to
