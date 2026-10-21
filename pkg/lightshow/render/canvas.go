// Package render draws the show primitives of the original iCubeSmart engine into voxel buffers.
package render

// Voxel colors are 6-bit 0bRRGGBB values.
const (
	ModeSet = 0 // overwrite
	ModeMix = 1 // bitwise OR into the existing voxel
)

// DefaultPalette is used when a show element does not name a palette.
const DefaultPalette = "\x00\x03\x0C\x0F\x30\x33\x3C\x3F\x00\x01\x04\x05\x10\x11\x14\x15"

// Display is one physical cube placed in the shared show coordinate space.
// Voxel index: x + y*L + z*L*H, where y is the vertical (layer) axis.
type Display struct {
	X, Y, Z int
	L, H, W int
	Voxels  []byte
}

func NewDisplay(x, y, z, l, h, w int) Display {
	return Display{X: x, Y: y, Z: z, L: l, H: h, W: w, Voxels: make([]byte, l*h*w)}
}

type Canvas struct {
	Displays []Display
}

func (this *Canvas) Clear(color byte) {
	for i := range this.Displays {
		v := this.Displays[i].Voxels
		for j := range v {
			v[j] = color
		}
	}
}

func put(dst *byte, color byte, mode int) {
	switch mode {
	case ModeSet:
		*dst = color
	case ModeMix:
		*dst |= color
	}
}

func paletteColor(palette string, index byte) byte {
	if int(index) < len(palette) {
		return palette[index]
	}
	return 0
}

// Point sets one voxel (script function Point). Any non-zero mode mixes.
func (this *Canvas) Point(x, y, z int, color byte, mix int) {
	for i := range this.Displays {
		d := &this.Displays[i]
		dx, dy, dz := x-d.X, y-d.Y, z-d.Z
		if dx < 0 || dx >= d.L || dy < 0 || dy >= d.H || dz < 0 || dz >= d.W {
			continue
		}
		p := &d.Voxels[dx+dy*d.L+dz*d.L*d.H]
		if mix == 0 {
			*p = color
		} else {
			*p |= color
		}
	}
}

// extend lengthens a coordinate delta by one voxel away from zero (as the vendor code does).
func extend(d int) int {
	if d < 1 {
		return d - 1
	}
	return d + 1
}

// Line draws a 3-D DDA line (script function Line). Any non-zero mode mixes.
func (this *Canvas) Line(x1, y1, z1, x2, y2, z2 int, color byte, mix int) {
	for i := range this.Displays {
		this.Displays[i].line(x1, y1, z1, x2, y2, z2, color, mix)
	}
}

func (d *Display) line(x1, y1, z1, x2, y2, z2 int, color byte, mix int) {
	x, y, z := x1-d.X, y1-d.Y, z1-d.Z
	lx := extend((x2 - d.X) - x)
	ly := extend((y2 - d.Y) - y)
	lz := extend((z2 - d.Z) - z)
	plot := func(px, py, pz int) {
		if px < 0 || px >= d.L || py < 0 || py >= d.H || pz < 0 || pz >= d.W {
			return
		}
		p := &d.Voxels[px+py*d.L+pz*d.L*d.H]
		if mix == 0 {
			*p = color
		} else {
			*p |= color
		}
	}
	ax, ay, az := lx*lx, ly*ly, lz*lz
	switch {
	case ax >= ay && ax >= az:
		dir, n := 1, lx
		if lx < 1 {
			dir, n = -1, -lx
		}
		end := (x2 - d.X) + dir
		for k := 0; x != end; k++ {
			plot(x, (k*ly)/n+y, (k*lz)/n+z)
			x += dir
		}
	case ay >= ax && ay >= az:
		dir, n := 1, ly
		if ly < 1 {
			dir, n = -1, -ly
		}
		end := (y2 - d.Y) + dir
		for k := 0; y != end; k++ {
			plot((k*lx)/n+x, y, (k*lz)/n+z)
			y += dir
		}
	default:
		dir, n := 1, lz
		if lz < 1 {
			dir, n = -1, -lz
		}
		end := (z2 - d.Z) + dir
		for k := 0; z != end; k++ {
			plot((k*lx)/n+x, (k*ly)/n+y, z)
			z += dir
		}
	}
}
