package network

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"slices"
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

	return this.write(UartMessage, []byte(message))
}

func (this *UartProcessor) WriteBytes(data []byte) error {
	if err := this.synchronize(); err != nil {
		return err
	}

	return this.write(UartBytes, data)
}

func (this *UartProcessor) SendPing() {
	this.write(UartPing, nil)
}

func (this *UartProcessor) SendPong() {
	this.write(UartPong, nil)
}

func (this *UartProcessor) Desynchronize() {
	this.synched = false
}

func (this *UartProcessor) Synchronize() error {
	return this.synchronize()
}

// Calls internal Read method, updates contentLength and
// if an error occurs, clears the buffer before returning
func (this *UartProcessor) read() (int, error) {
	// TODO: check if this is correct or is this.buffer[this.bufferIndex:] needed
	n, err := this.uart.Read(this.buffer)
	this.contentLength += n
	if err != nil {
		this.clearBuffer(this.contentLength)
	}
	return n, err
}

func (this *UartProcessor) write(dType UartDataType, payload []byte) error {
	if payloadLength := len(payload); maxPayloadSize >= payloadLength {
		_, err := this.uart.Write(payloadToUartPacket(payload, dType))
		return err
	}

	currentPayload, nextPayload := splitPayload(payload, dType)
	if _, err := this.uart.Write(payloadToUartPacket(currentPayload, dType)); err != nil {
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
