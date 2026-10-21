package render

// Voxels is a decoded graph: palette indexes, x fastest, then y, then z.
type Voxels struct {
	L, H, W int
	Data    []byte
}

// sinTable holds sin(0..90 degrees) * 1000, as in the vendor engine.
var sinTable = [91]int16{
	0, 17, 34, 52, 69, 87, 104, 121, 139, 156, 173, 190, 207, 224, 241, 258, 275, 292, 309, 325,
	342, 358, 374, 390, 406, 422, 438, 453, 469, 484, 500, 515, 529, 544, 559, 573, 587, 601, 615, 629,
	642, 656, 669, 681, 694, 707, 719, 731, 743, 754, 766, 777, 788, 798, 809, 819, 829, 838, 848, 857,
	866, 874, 882, 891, 898, 906, 913, 920, 927, 933, 939, 945, 951, 956, 961, 965, 970, 974, 978, 981,
	984, 987, 990, 992, 994, 996, 997, 998, 999, 999, 1000,
}

// sinCos returns sin and cos * 1000 for an angle in 0..359 degrees.
func sinCos(a int) (s, c int) {
	switch {
	case a < 91:
		return int(sinTable[a]), int(sinTable[90-a])
	case a < 181:
		return int(sinTable[180-a]), -int(sinTable[a-90])
	case a < 271:
		return -int(sinTable[a-180]), -int(sinTable[270-a])
	default:
		return -int(sinTable[360-a]), int(sinTable[a-270])
	}
}

// Rotation axes
const (
	AxisX = 0
	AxisY = 1
	AxisZ = 2
)

// Graph draws a graph with its minimum corner at (x, y, z), rotated by angle degrees
// around the given axis (through the graph centre). palette maps the graph's indexes to colors.
func (this *Canvas) Graph(g Voxels, palette string, x, y, z, axis, angle, mode int) {
	angle %= 360
	if angle < 0 {
		angle += 360
	}
	for i := range this.Displays {
		d := &this.Displays[i]
		if angle == 0 {
			d.blit(g, palette, x, y, z, mode)
		} else {
			d.rotated(g, palette, x, y, z, axis, angle, mode)
		}
	}
}

func (d *Display) blit(g Voxels, palette string, gx, gy, gz, mode int) {
	if gx >= d.X+d.L || gy >= d.Y+d.H || gz >= d.Z+d.W {
		return
	}
	xs, sx := clipStart(gx, d.X)
	ys, sy := clipStart(gy, d.Y)
	zs, sz := clipStart(gz, d.Z)
	xe := min(g.L-d.X+gx, d.L)
	ye := min(g.H-d.Y+gy, d.H)
	ze := min(g.W-d.Z+gz, d.W)
	nx, ny, nz := xe-xs, ye-ys, ze-zs
	if nx <= 0 || ny <= 0 || nz <= 0 {
		return
	}
	for k := 0; k < nz; k++ {
		for j := 0; j < ny; j++ {
			dst := xs + (ys+j)*d.L + (zs+k)*d.L*d.H
			src := sx + (sy+j)*g.L + (sz+k)*g.L*g.H
			for i := 0; i < nx; i++ {
				put(&d.Voxels[dst+i], paletteColor(palette, g.Data[src+i]), mode)
			}
		}
	}
}

// clipStart returns the first covered display offset and the matching graph offset.
func clipStart(g, origin int) (displayStart, graphStart int) {
	if g < origin {
		return 0, origin - g
	}
	return g - origin, 0
}

// rotated ports the vendor's slice rotation: every line of voxels along the axis is moved
// as a whole to its rotated position in the plane perpendicular to the axis.
func (d *Display) rotated(g Voxels, palette string, gx, gy, gz, axis, angle, mode int) {
	s, c := sinCos(angle)
	var (
		n1, n2         int // plane size: i loops over n1, j over n2
		base1, base2   int // plane origin in show coordinates
		lo1, lo2       int // display range of the two plane coordinates
		hi1, hi2       int
		runStart, runE int // clipped run along the axis, show coordinates
	)
	switch axis {
	case AxisX:
		n1, n2, base1, base2 = g.W, g.H, gz, gy
		lo1, hi1, lo2, hi2 = d.Z, d.Z+d.W, d.Y, d.Y+d.H
		runStart, runE = max(gx, d.X), min(gx+g.L, d.X+d.L)
	case AxisY:
		n1, n2, base1, base2 = g.L, g.W, gx, gz
		lo1, hi1, lo2, hi2 = d.X, d.X+d.L, d.Z, d.Z+d.W
		runStart, runE = max(gy, d.Y), min(gy+g.H, d.Y+d.H)
	case AxisZ:
		n1, n2, base1, base2 = g.L, g.H, gx, gy
		lo1, hi1, lo2, hi2 = d.X, d.X+d.L, d.Y, d.Y+d.H
		runStart, runE = max(gz, d.Z), min(gz+g.W, d.Z+d.W)
	default:
		return
	}
	if runStart >= runE {
		return
	}
	lh, glh := d.L*d.H, g.L*g.H
	for i := 0; i < n1; i++ {
		u := 100*i - 50*n1
		for j := 0; j < n2; j++ {
			v := 100*j - 50*n2
			p1 := (c*u+s*v)/100000 + n1/2 + base1
			p2 := (c*v-s*u)/100000 + n2/2 + base2
			if p1 < lo1 || p1 >= hi1 || p2 < lo2 || p2 >= hi2 {
				continue
			}
			var src, srcStep, dst, dstStep int
			switch axis {
			case AxisX: // i = graph z, j = graph y; p1 = z, p2 = y
				src, srcStep = (runStart-gx)+j*g.L+i*glh, 1
				dst, dstStep = (runStart-d.X)+(p2-d.Y)*d.L+(p1-d.Z)*lh, 1
			case AxisY: // i = graph x, j = graph z; p1 = x, p2 = z
				src, srcStep = i+(runStart-gy)*g.L+j*glh, g.L
				dst, dstStep = (p1-d.X)+(runStart-d.Y)*d.L+(p2-d.Z)*lh, d.L
			case AxisZ: // i = graph x, j = graph y; p1 = x, p2 = y
				src, srcStep = i+j*g.L+(runStart-gz)*glh, glh
				dst, dstStep = (p1-d.X)+(p2-d.Y)*d.L+(runStart-d.Z)*lh, lh
			}
			for k := runStart; k < runE; k++ {
				put(&d.Voxels[dst], paletteColor(palette, g.Data[src]), mode)
				src += srcStep
				dst += dstStep
			}
		}
	}
}

// Ball draws a sphere. Centre and radius are in 1/100 voxel units. The outer 0.9 voxel shell
// gets colorOut, the next shell colorIn, and the core colorIn unless hollow.
func (this *Canvas) Ball(x, y, z, r int, colorOut, colorIn byte, hollow bool) {
	for i := range this.Displays {
		this.Displays[i].ball(x, y, z, r, colorOut, colorIn, hollow)
	}
}

func (d *Display) ball(x, y, z, r int, colorOut, colorIn byte, hollow bool) {
	cx, cy, cz := x-d.X*100, y-d.Y*100, z-d.Z*100
	x0, x1 := max((cx-r)/100, 0), min((r+100+cx)/100, d.L)
	y0, y1 := max((cy-r)/100, 0), min((cy+r+100)/100, d.H)
	z0, z1 := max((cz-r)/100, 0), min((cz+r+100)/100, d.W)
	rr := uint32(r * r)
	outer := (r - 90) * (r - 90)
	inner := (r - 180) * (r - 180)
	for vz := z0; vz < z1; vz++ {
		dz := vz*100 - cz
		for vx := x0; vx < x1; vx++ {
			dx := vx*100 - cx
			for vy := y0; vy < y1; vy++ {
				dy := vy*100 - cy
				dist := dy*dy + dz*dz + dx*dx
				if uint32(dist) > rr {
					continue
				}
				p := &d.Voxels[vx+vy*d.L+vz*d.L*d.H]
				switch {
				case outer < dist:
					*p = colorOut
				case inner < dist:
					*p = colorIn
				case !hollow:
					*p = colorIn
				}
			}
		}
	}
}

// Title scrolls a text strip along a path. path holds (i16 x, i16 z) column positions;
// column 0 of the strip is at path[step] and later columns trail behind it.
func (this *Canvas) Title(g Voxels, palette string, path string, high, style, step int) {
	for i := range this.Displays {
		this.Displays[i].title(g, palette, path, high, style, step)
	}
}

func (d *Display) title(g Voxels, palette string, path string, high, style, step int) {
	yOff := high - d.Y
	columns := g.W * g.L
	points := len(path) / 4
	if step < 0 || points == 0 || yOff >= d.H || g.H+yOff <= 0 {
		return
	}
	var dy, gy0 int
	if yOff < 1 {
		gy0 = -yOff
	} else {
		dy = yOff
	}
	gy1 := g.H
	if d.H < (g.H-gy0)+dy {
		gy1 = d.H - dy + gy0
	}
	if step >= points-1+columns {
		return
	}
	low := -1
	if columns <= step {
		low = step - columns
	}
	col := 0
	if points <= step {
		col = step - points + 1
		step = points - 1
	}
	for p := step; p > low; p-- {
		px := pathValue(path, 2*p) - d.X
		pz := pathValue(path, 2*p+1) - d.Z
		if px >= 0 && px < d.L && pz >= 0 && pz < d.W {
			var src int
			switch style {
			case 0:
				src = (g.H*(col/g.L)+gy0)*g.L + col%g.L
			case 1:
				src = (gy0+1+g.H*(col/g.L))*g.L - col%g.L - 1
			default:
				src = -1
			}
			if src >= 0 {
				dst := dy*d.L + pz*d.L*d.H + px
				for y := gy0; y < gy1; y++ {
					d.Voxels[dst] = paletteColor(palette, g.Data[src])
					dst += d.L
					src += g.L
				}
			}
		}
		col++
	}
}

func pathValue(path string, i int) int {
	return int(int16(uint16(path[2*i]) | uint16(path[2*i+1])<<8))
}
