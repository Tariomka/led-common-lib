package network

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"slices"
)

var (
	ErrHandshake     = errors.New("handshake failed")
	ErrExpectedStart = errors.New("expected start marker")
	ErrExpectedEnd   = errors.New("expected end marker")
	ErrTooMuchData   = errors.New("too much data")
	ErrChecksum      = errors.New("checksum error")
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
	startMarker = byte(0xAA)
	endMarker   = byte(0x55)

	maxRetries = 3

	bufferSize        = 1024
	sizeBeforeContent = 1 + 1 + 2 // start marker + type + content length
	checksumSize      = 4
	sizeAfterContent  = checksumSize + 1 // checksum + end marker
	nonContentSize    = sizeBeforeContent + sizeAfterContent
	maxPayloadSize    = bufferSize - nonContentSize
)

type UartProcessor struct {
	buffer        []byte // TODO: think about a [bufferSize]byte array instead of slice, to not allocate
	uart          io.ReadWriter
	contentLength int
	synched       bool
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

	for this.contentLength < sizeBeforeContent {
		n, err := this.read()
		if err != nil {
			return UartEmpty, nil, err
		}
		if n == 0 {
			this.clearBuffer(this.contentLength)
			return UartEmpty, nil, nil
		}
	}

	if this.buffer[0] != startMarker {
		this.clearBuffer(this.contentLength)
		return UartEmpty, nil, ErrExpectedStart
	}

	dType := UartDataType(this.buffer[1])
	payloadSize := binary.BigEndian.Uint16(this.buffer[2:4])
	if payloadSize > maxPayloadSize {
		this.clearBuffer(this.contentLength)
		return UartEmpty, nil, ErrTooMuchData
	}

	packetSize := nonContentSize + int(payloadSize)
	for this.contentLength < packetSize {
		n, err := this.read()
		if err != nil {
			return UartEmpty, nil, err
		}
		if n == 0 {
			this.clearBuffer(this.contentLength)
			return UartEmpty, nil, nil
		}
	}

	defer this.clearBuffer(packetSize)
	if this.buffer[packetSize-1] != endMarker {
		return UartEmpty, nil, ErrExpectedEnd
	}

	content := this.buffer[sizeBeforeContent : sizeBeforeContent+payloadSize]
	checksumPos := sizeBeforeContent + payloadSize
	if checksum := binary.BigEndian.Uint32(
		this.buffer[checksumPos : checksumPos+checksumSize]); checksum != crc32.ChecksumIEEE(content) {
		return UartEmpty, nil, ErrChecksum
	}

	return dType, append([]byte{}, content...), nil
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

// Calls internal Read method, updates contentLength and if an error occurs, clears the buffer before returning
func (this *UartProcessor) read() (int, error) {
	// n, err := this.uart.Read(this.buffer[this.bufferIndex:])
	n, err := this.uart.Read(this.buffer)
	this.contentLength += n
	if err != nil {
		this.clearBuffer(this.contentLength)
	}
	return n, err
}

func (this *UartProcessor) write(dType UartDataType, payload []byte) error {
	if payloadLength := len(payload); maxPayloadSize >= payloadLength {
		_, err := this.uart.Write(payloadToPacket(payload, dType))
		return err
	}

	currentPayload, nextPayload := splitPayload(payload, dType)
	if _, err := this.uart.Write(payloadToPacket(currentPayload, dType)); err != nil {
		return err
	}

	return this.write(dType, nextPayload)
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

	retries := 0
	for this.contentLength < handshakeLength {
		_, err := this.read()
		if err != nil || retries >= maxRetries {
			return false
		}

		retries++
	}

	if len(this.buffer) < handshakeLength ||
		slices.Compare(this.buffer[:handshakeLength], handshake) != 0 {
		this.clearBuffer(this.contentLength)
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
		this.contentLength = 0
		return
	}

	clear(this.buffer[:index])
	this.buffer = append(this.buffer[index:], this.buffer[:index]...)[:bufferSize:bufferSize]
	if this.contentLength >= index {
		this.contentLength -= index
	} else {
		this.contentLength = 0
	}
}

// TODO: add a function for formatting strings to messages for
// easy sending with print(createMessage("message"))
// probably should be something like:
// func CreateUartMessage(message string) string
// and most likely will need to

func payloadToPacket(payload []byte, dType UartDataType) []byte {
	var buffer bytes.Buffer
	buffer.WriteByte(startMarker)
	buffer.WriteByte(byte(dType))

	lengthBytes := [2]byte{}
	binary.BigEndian.PutUint16(lengthBytes[:], uint16(len(payload)))
	buffer.Write(lengthBytes[:])
	buffer.Write(payload)

	checksum := [4]byte{}
	binary.BigEndian.PutUint32(checksum[:], crc32.ChecksumIEEE(payload))
	buffer.Write(checksum[:])
	buffer.WriteByte(endMarker)

	return buffer.Bytes()
}

func splitPayload(payload []byte, dType UartDataType) (current []byte, next []byte) {
	if len(payload) <= maxPayloadSize {
		return payload, nil
	}

	if dType == UartMessage {
		payloadLength := uint16(maxPayloadSize - 3)
		return slices.Concat(payload[:payloadLength], ellipsis),
			slices.Concat(ellipsis, payload[payloadLength:])
	}

	return payload[:maxPayloadSize], payload[maxPayloadSize:]
}
