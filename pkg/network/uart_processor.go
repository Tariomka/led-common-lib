package network

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"slices"
	"time"
)

type UartProcessor struct {
	buffer        [bufferSize]byte
	uart          io.ReadWriter
	contentLength int
	synched       bool
}

func NewUartProcessor(uart io.ReadWriter) *UartProcessor {
	return &UartProcessor{
		uart: uart,
	}
}

func (this *UartProcessor) Read() (dType UartDataType, content []byte, err error) {
	if err = this.synchronize(); err != nil {
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

	dType = UartDataType(this.buffer[1])
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

	content = this.buffer[sizeBeforeContent : sizeBeforeContent+payloadSize]
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

// ReadWithoutProcessing receives data dirctly without handshake or processing.
// This is primarily used for debugging.
func (this *UartProcessor) ReadWithoutProcessing() ([]byte, error) {
	length, err := this.uart.Read(this.buffer[this.contentLength:])
	this.contentLength += length
	defer this.clearBuffer(this.contentLength)
	return append([]byte{}, this.buffer[:this.contentLength]...), err
}

// Calls internal Read method, updates contentLength and
// if an error occurs, clears the buffer before returning
func (this *UartProcessor) read() (length int, err error) {
	length, err = this.uart.Read(this.buffer[this.contentLength:])
	this.contentLength += length
	if err != nil {
		this.clearBuffer(this.contentLength)
	}
	return length, err
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

// Blocks until handshake is established or timeout occurs.
// Returns true if handshake is successful and removes handshake from buffer,
// false otherwise and cleans buffer completely.
func (this *UartProcessor) establishHandshake() bool {
	if n, err := this.uart.Write(handshake); err != nil || n != handshakeLength {
		return false
	}

	timeoutTimer := time.After(handshakeTimeout)
	isFirstRead := true
	for this.contentLength < handshakeLength {
		select {
		case <-timeoutTimer:
			return false
		default:
			if _, err := this.read(); err != nil {
				return false
			}

			if index := this.searchForHandshake(); index > 0 {
				this.clearBuffer(index)
			}

			if isFirstRead {
				isFirstRead = false
			} else {
				<-time.After(handshakeDelay)
			}
		}
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
		clear(this.buffer[:])
		this.contentLength = 0
		return
	}

	clear(this.buffer[:index])
	for i := index; i < bufferSize; i++ {
		this.buffer[i-index] = this.buffer[i]
	}
	if this.contentLength >= index {
		this.contentLength -= index
	} else {
		this.contentLength = 0
	}
}

func (this *UartProcessor) searchForHandshake() int {
	startIndex := 0
	for startIndex = range this.contentLength - handshakeLength + 1 {
		if slices.Compare(this.buffer[startIndex:startIndex+handshakeLength], handshake) == 0 {
			return startIndex
		}
	}

	maxLength := this.contentLength - startIndex
	for continuationIndex := range maxLength {
		if slices.Compare(this.buffer[startIndex+continuationIndex:this.contentLength], handshake[:maxLength-continuationIndex]) == 0 {
			return startIndex + continuationIndex
		}
	}

	return this.contentLength
}
