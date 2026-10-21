package keyframe

import "github.com/Tariomka/led-common-lib/pkg/lightshow/render"

// Shape is a voxel picture of palette indexes. Art has one line per layer from the top
// (y = H-1) down to the bottom (y = 0); a line holds W rows (z = 0, 1, ...) separated by
// spaces, each with L voxels (x = 0, 1, ...). A voxel is '.' (index 0) or a hex digit 1..F.
type Shape struct {
	Name    string
	L, H, W int
	Art     string
}

// Palette maps the 16 shape indexes to colors (0bRRGGBB).
type Palette struct {
	Name   string
	Colors string
}

// Library holds the shapes and palettes a show refers to by name.
type Library struct {
	Shapes   []Shape
	Palettes []Palette
	// Missing reports unknown names (optional).
	Missing func(kind, name string)

	buf    []byte
	cached string
	voxels render.Voxels
}

func (this *Library) missing(kind, name string) {
	if this.Missing != nil {
		this.Missing(kind, name)
	}
}

// shape decodes a shape's art; the result is valid until the next call.
func (this *Library) shape(name string) (render.Voxels, bool) {
	if name == this.cached && name != "" {
		return this.voxels, true
	}
	for i := range this.Shapes {
		shape := &this.Shapes[i]
		if shape.Name != name {
			continue
		}
		size := shape.L * shape.H * shape.W
		if cap(this.buf) < size {
			this.buf = make([]byte, size)
		}
		data := this.buf[:size]
		if !decodeArt(shape, data) {
			this.missing("shape art", name)
			return render.Voxels{}, false
		}
		this.cached = name
		this.voxels = render.Voxels{L: shape.L, H: shape.H, W: shape.W, Data: data}
		return this.voxels, true
	}
	this.missing("shape", name)
	return render.Voxels{}, false
}

// decodeArt fills data (x fastest, then y, then z) from the art text.
func decodeArt(shape *Shape, data []byte) bool {
	i, n := 0, len(data)
	for j := 0; j < len(shape.Art); j++ {
		ch := shape.Art[j]
		var value byte
		switch {
		case ch == '.' || ch == '0':
		case ch >= '1' && ch <= '9':
			value = ch - '0'
		case ch >= 'A' && ch <= 'F':
			value = ch - 'A' + 10
		default:
			continue
		}
		if i >= n {
			return false
		}
		x := i % shape.L
		z := i / shape.L % shape.W
		y := shape.H - 1 - i/(shape.L*shape.W)
		data[x+y*shape.L+z*shape.L*shape.H] = value
		i++
	}
	return i == n
}

func (this *Library) palette(name string) (string, bool) {
	if name == "" {
		return render.DefaultPalette, true
	}
	for i := range this.Palettes {
		if this.Palettes[i].Name == name {
			return this.Palettes[i].Colors, true
		}
	}
	this.missing("palette", name)
	return "", false
}
