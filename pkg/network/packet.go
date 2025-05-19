package network

import (
	"encoding/binary"

	"github.com/Tariomka/led-common-lib/pkg/common"
)

type DataType byte

const (
	Message DataType = iota
	RGB8x8
	RGB16x16
	RGB8x32
	Mono8x8
	Setting
)

type Version byte

const (
	V1 Version = iota + 1
	v2
)

const (
	headerSize  = 2
	settingSize = 4

	ByteSize8x8     = 192
	ByteSize8x8Mono = 64
	ByteSize16x16   = 768
	ByteSize8x32    = 768
)

type Packet struct {
	Version Version
	Type    DataType
	Data    []byte
}

func NewMessagePacket(data string) Packet {
	return Packet{
		Version: V1,
		Type:    Message,
		Data:    []byte(data),
	}
}

func NewLedPacket(dtype DataType, data []byte) Packet {
	return Packet{
		Version: V1,
		Type:    dtype,
		Data:    data,
	}
}

func NewSettingPacket(setting, value uint16) Packet {
	data := []byte{}
	data = binary.BigEndian.AppendUint16(data, setting)
	data = binary.BigEndian.AppendUint16(data, value)
	// The underlying bytes look like [s1 s0 v1 v0]
	// Where s0 - 0-255 and goes up from there
	return Packet{
		Version: V1,
		Type:    Setting,
		Data:    data,
	}
}

func (p Packet) Marshall() []byte {
	bytes := make([]byte, 0, 1+1+len(p.Data))
	bytes = append(bytes, byte(p.Version), byte(p.Type))
	bytes = append(bytes, p.Data...)
	return bytes
}

func UnmarshallPacket(data []byte) (*Packet, error) {
	if len(data) < headerSize {
		return nil, common.ErrNotEnoughData
	}

	version := Version(data[0])
	if version != V1 {
		return nil, common.ErrUnsupportedVersion
	}

	dtype := DataType(data[1])

	packet := &Packet{
		Version: version,
		Type:    dtype,
	}

	switch dtype {
	case RGB8x8:
		packet.Data = data[headerSize : ByteSize8x8+headerSize]
	case RGB16x16:
		packet.Data = data[headerSize : ByteSize16x16+headerSize]
	case RGB8x32:
		packet.Data = data[headerSize : ByteSize8x32+headerSize]
	case Mono8x8:
		packet.Data = data[headerSize : ByteSize8x8Mono+headerSize]
	case Message:
		if end := common.FindFirstIndex(data[headerSize:], 0); end > -1 {
			packet.Data = data[headerSize : end+headerSize]
		} else {
			packet.Data = data[headerSize:] // Maybe can be nil?
		}
	case Setting:
		packet.Data = data[headerSize : settingSize+headerSize]
	default:
		packet.Data = data[headerSize:]
	}

	return packet, nil
}
