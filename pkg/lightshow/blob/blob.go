// Package blob reads the show data ("CartoonDatas") produced by the iCubeSmart PC software.
// The data is kept as a string so data embedded in a firmware image stays in flash on the microcontroller.
package blob

import "github.com/Tariomka/led-common-lib/pkg/common"

const (
	Magic      = 0x12348765
	headerSize = 68
	nameSize   = 16

	PaletteSize = 16
)

const (
	secConfig = iota
	secPalette
	secGraph
	secIndex
	secGroup1
	GroupCount = 4
)

type Blob struct {
	data string
	sec  [8]section
}

func Parse(data string) (*Blob, error) {
	if len(data) < headerSize || U32(data, 0) != Magic {
		return nil, common.ErrBlobMagic
	}

	blob := &Blob{data: data}
	for index := range blob.sec {
		off, n := int(U32(data, 4+8*index)), int(U32(data, 8+8*index))
		if off < headerSize || off+n > len(data) {
			return nil, common.ErrBlobLayout
		}

		blob.sec[index] = section{off, n}
	}
	return blob, nil
}

// Size is the used length of the blob in bytes.
func (this *Blob) Size() int {
	last := this.sec[len(this.sec)-1]
	return last.off + last.len
}

// KeyFunction returns the configured action for KEY1..KEY7 (key 0..6).
func (this *Blob) KeyFunction(key int) int {
	if key < 0 || key > 6 {
		return 0
	}

	return int(this.data[this.sec[secConfig].off+key])
}

func (this *Blob) DisplayCount() int {
	return int(this.data[this.sec[secConfig].off+7])
}

func (this *Blob) Display(index int) Display {
	offset := this.sec[secConfig].off + 8 + 8*index
	return Display{
		X: I16(this.data, offset), Y: I16(this.data, offset+2), Z: I16(this.data, offset+4),
		Dir:      int(this.data[offset+6]),
		CubeType: int(this.data[offset+7]),
	}
}

// Palette returns the 16 color bytes of a named palette.
func (this *Blob) Palette(name string) (string, bool) {
	section := this.sec[secPalette]
	for offset := section.off; offset+32 <= section.off+section.len; offset += 32 {
		if Name(this.data, offset) == name {
			return this.data[offset+nameSize : offset+32], true
		}
	}

	return "", false
}

func (this *Blob) graphAt(offset int) Graph {
	dl := int(U32(this.data, offset+20))
	return Graph{
		Name:   Name(this.data, offset),
		Length: int(this.data[offset+24]), Height: int(this.data[offset+25]), Width: int(this.data[offset+26]),
		Compression: int(this.data[offset+27]),
		Data:        this.data[offset+28 : offset+28+dl],
		offset:      offset,
	}
}

func (this *Blob) Graph(name string) (Graph, bool) {
	section := this.sec[secGraph]
	for offset := section.off; offset+28 <= section.off+section.len; {
		size := int(U32(this.data, offset+16))
		if size <= 0 {
			break
		}

		if Name(this.data, offset) == name {
			return this.graphAt(offset), true
		}

		offset += size
	}
	return Graph{}, false
}

// ForEachGraph visits all graphs in blob order.
func (this *Blob) ForEachGraph(callback func(Graph)) {
	section := this.sec[secGraph]
	for offset := section.off; offset+28 <= section.off+section.len; {
		size := int(U32(this.data, offset+16))
		if size <= 0 {
			return
		}

		callback(this.graphAt(offset))
		offset += size
	}
}

// Index returns the raw data of a named index table.
func (this *Blob) Index(name string) (string, bool) {
	section := this.sec[secIndex]
	for offset := section.off; offset+20 <= section.off+section.len; {
		n := int(U32(this.data, offset+16))
		if Name(this.data, offset) == name {
			return this.data[offset+20 : offset+20+n], true
		}

		offset += 20 + (n+3)&^3
	}
	return "", false
}

// Group returns all cartoons of group (0..3). Allocates once per call.
func (this *Blob) Group(group int) []Cartoon {
	if group < 0 || group >= GroupCount {
		return nil
	}

	section := this.sec[secGroup1+group]
	var list []Cartoon
	for offset := section.off; offset+8 <= section.off+section.len; {
		size := int(U32(this.data, offset))
		if size < 8 || offset+size > section.off+section.len {
			break
		}

		list = append(list, Cartoon{Type: int(this.data[offset+4]), Body: this.data[offset+8 : offset+size]})
		offset += size
	}
	return list
}
