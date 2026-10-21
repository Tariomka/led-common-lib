package blob

import "github.com/Tariomka/led-common-lib/pkg/common"

// MaxSize is the largest show data the cube can store.
const MaxSize = 160000

// Marshal compiles the show into show data. Graphs are stored with whichever compression is
// smallest.
func (this *Show) Marshal() ([]byte, error) {
	w := &writer{buf: make([]byte, headerSize, 4096)}
	w.putU32(0, Magic)

	start := len(w.buf)
	w.buf = append(w.buf, this.Config.Keys[:]...)
	w.u8(len(this.Config.Displays))
	for _, display := range this.Config.Displays {
		w.i16(display.X)
		w.i16(display.Y)
		w.i16(display.Z)
		w.u8(display.Dir)
		w.u8(display.CubeType)
	}
	w.section(secConfig, start)

	start = len(w.buf)
	for _, palette := range this.Palettes {
		w.name(palette.Name)
		w.buf = append(w.buf, palette.Colors[:]...)
	}
	w.section(secPalette, start)

	start = len(w.buf)
	for _, graph := range this.Graphs {
		w.graph(graph)
	}
	w.section(secGraph, start)

	start = len(w.buf)
	for _, index := range this.Indexes {
		w.name(index.Name)
		w.u32(uint32(len(index.Data)))
		w.buf = append(w.buf, index.Data...)
		w.pad4()
	}
	w.section(secIndex, start)

	for group, cartoons := range this.Groups {
		start = len(w.buf)
		for _, cartoon := range cartoons {
			w.cartoon(cartoon)
		}
		w.section(secGroup1+group, start)
	}

	if w.err != nil {
		return nil, w.err
	}
	if len(w.buf) > MaxSize {
		return nil, common.ErrBlobTooLarge
	}
	return w.buf, nil
}

type writer struct {
	buf []byte
	err error
}

func (this *writer) fail(err error) {
	if this.err == nil {
		this.err = err
	}
}

func (this *writer) u8(value int) {
	if int(uint8(value)) != value {
		this.fail(common.ErrBlobValue)
	}
	this.buf = append(this.buf, byte(value))
}

func (this *writer) u16(value int) {
	if int(uint16(value)) != value {
		this.fail(common.ErrBlobValue)
	}
	this.buf = append(this.buf, byte(value), byte(value>>8))
}

func (this *writer) i16(value int) {
	if int(int16(value)) != value {
		this.fail(common.ErrBlobValue)
	}
	this.buf = append(this.buf, byte(value), byte(value>>8))
}

// h16 writes a 16-bit field the engine reads either signed or unsigned.
func (this *writer) h16(value int) {
	if int(int16(value)) != value && int(uint16(value)) != value {
		this.fail(common.ErrBlobValue)
	}
	this.buf = append(this.buf, byte(value), byte(value>>8))
}

func (this *writer) i32(value int) {
	if int(int32(value)) != value {
		this.fail(common.ErrBlobValue)
	}
	this.u32(uint32(value))
}

func (this *writer) u32(value uint32) {
	this.buf = append(this.buf, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
}

func (this *writer) putU32(at int, value uint32) {
	this.buf[at], this.buf[at+1], this.buf[at+2], this.buf[at+3] = byte(value), byte(value>>8), byte(value>>16), byte(value>>24)
}

func (this *writer) name(name string) {
	if len(name) >= nameSize {
		this.fail(common.ErrBlobNameTooLong)
		name = name[:nameSize-1]
	}
	this.buf = append(this.buf, name...)
	for i := len(name); i < nameSize; i++ {
		this.buf = append(this.buf, 0)
	}
}

func (this *writer) pad4() {
	for len(this.buf)%4 != 0 {
		this.buf = append(this.buf, 0)
	}
}

func (this *writer) section(index, start int) {
	this.putU32(4+8*index, uint32(start))
	this.putU32(8+8*index, uint32(len(this.buf)-start))
}

func (this *writer) graph(graph GraphData) {
	if graph.L < 1 || graph.L > 255 || graph.H < 1 || graph.H > 255 || graph.W < 1 || graph.W > 255 ||
		len(graph.Voxels) != graph.L*graph.H*graph.W {
		this.fail(common.ErrBlobValue)
		return
	}

	data, compression := encodeGraph(graph.Voxels)
	this.name(graph.Name)
	this.u32(uint32(28 + (len(data)+3)&^3))
	this.u32(uint32(len(data)))
	this.buf = append(this.buf, byte(graph.L), byte(graph.H), byte(graph.W), byte(compression))
	this.buf = append(this.buf, data...)
	this.pad4()
}

func (this *writer) cartoon(cartoon CartoonData) {
	start := len(this.buf)
	this.u32(0)
	this.u8(cartoon.Type)
	this.buf = append(this.buf, 0, 0, 0)
	if cartoon.Type == Normal || cartoon.Type == Audio {
		this.i32(cartoon.Header.FrameCount)
		this.h16(cartoon.Header.Loop)
		this.u16(cartoon.Header.Delay)
		for _, element := range cartoon.Elements {
			this.element(element)
		}
	} else {
		for _, word := range cartoon.Script {
			this.u32(word)
		}
	}
	this.putU32(start, uint32(len(this.buf)-start))
}

func (this *writer) element(element ElementData) {
	start := len(this.buf)
	this.u32(0)
	this.u32(uint32(element.Kind))
	switch element.Kind {
	case SetGraph:
		for _, prop := range element.Graphs {
			this.i16(prop.Start)
			this.i16(prop.End)
			this.name(prop.Graph)
			this.name(prop.Palette)
			for _, value := range [...]int{prop.X0, prop.X1, prop.Y0, prop.Y1, prop.Z0, prop.Z1, prop.Angle0, prop.Angle1} {
				this.i16(value)
			}
			this.h16(prop.Axis)
			this.h16(prop.Mode)
		}
	case DrawBall:
		for _, prop := range element.Balls {
			for _, value := range [...]int{prop.Start, prop.End, prop.X, prop.Y, prop.Z, prop.R0, prop.R1} {
				this.i16(value)
			}
			hollow := 0
			if prop.Hollow {
				hollow = 1
			}
			this.i16(hollow)
			this.buf = append(this.buf, prop.ColorIn, prop.ColorOut, 0, 0)
		}
	case TitleStyle:
		for _, prop := range element.Titles {
			this.i16(prop.Start)
			this.i16(prop.End)
			this.name(prop.Graph)
			this.name(prop.Palette)
			this.name(prop.Index)
			this.i16(prop.High)
			this.h16(prop.Style)
		}
	default:
		this.fail(common.ErrBlobValue)
	}
	this.putU32(start, uint32(len(this.buf)-start))
}

// encodeGraph tries every compression and returns the smallest encoding.
func encodeGraph(voxels []byte) ([]byte, int) {
	best, compression := voxels, NoCompress

	var used [256]bool
	for _, value := range voxels {
		used[value] = true
	}
	var lut []byte
	var position [256]byte
	for value := range used {
		if used[value] {
			position[value] = byte(len(lut))
			lut = append(lut, byte(value))
		}
	}

	for i, bits := range [3]uint{4, 2, 1} {
		if len(lut) > 1<<bits {
			continue
		}
		if data := encodePacked(voxels, lut, &position, bits); len(data) < len(best) {
			best, compression = data, Normal2+i
		}
		if data := encodeRLE(voxels, lut, &position, bits); len(data) < len(best) {
			best, compression = data, Continue2+i
		}
	}
	return best, compression
}

func encodePacked(voxels, lut []byte, position *[256]byte, bits uint) []byte {
	perByte := 8 / int(bits)
	data := make([]byte, 0, 1+len(lut)+(len(voxels)+perByte-1)/perByte)
	data = append(append(data, byte(len(lut))), lut...)
	for i := 0; i < len(voxels); i += perByte {
		var packed byte
		for j := 0; j < perByte && i+j < len(voxels); j++ {
			packed |= position[voxels[i+j]] << (uint(j) * bits)
		}
		data = append(data, packed)
	}
	return data
}

func encodeRLE(voxels, lut []byte, position *[256]byte, bits uint) []byte {
	data := append([]byte{byte(len(lut))}, lut...)
	maxCount := 1<<(15-bits) - 1
	for i := 0; i < len(voxels); {
		value, count := voxels[i], 1
		for i+count < len(voxels) && voxels[i+count] == value && count < maxCount {
			count++
		}
		i += count

		index := position[value]
		if count < 1<<(7-bits) {
			data = append(data, byte(count<<bits)|index)
		} else {
			data = append(data, 0x80|byte(count>>(8-bits)), byte(count<<bits)|index)
		}
	}
	return data
}
