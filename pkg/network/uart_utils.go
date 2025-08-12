package network

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"slices"
	"time"
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
var handshakeLength = len(handshake)   // 12
var ellipsis = []byte("...")

const (
	startMarker = byte(0xAA)
	endMarker   = byte(0x55)

	handshakeTimeout = 15 * time.Second
	handshakeDelay   = 500 * time.Millisecond

	bufferSize        = 1024
	sizeBeforeContent = 1 + 1 + 2 // start marker + type + content length
	checksumSize      = 4
	sizeAfterContent  = checksumSize + 1 // checksum + end marker
	nonContentSize    = sizeBeforeContent + sizeAfterContent
	maxPayloadSize    = bufferSize - nonContentSize
)

// ToUartMessage creates a byte slice with formated UART message packet
func ToUartMessage(message string) []byte {
	return payloadToUartPacket([]byte(message), UartMessage)
}

func payloadToUartPacket(payload []byte, dType UartDataType) []byte {
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
