package blob

import "github.com/Tariomka/led-common-lib/pkg/common"

// Show is the editable form of show data: Marshal compiles it, FromBlob reads it back.
type Show struct {
	Config   Config
	Palettes []Palette
	Graphs   []GraphData
	Indexes  []IndexData
	Groups   [GroupCount][]CartoonData
}

type Config struct {
	Keys     [7]byte // key function codes of KEY1..KEY7
	Displays []Display
}

type Palette struct {
	Name   string
	Colors [PaletteSize]byte // 0bRRGGBB
}

// GraphData is a graph with decoded voxels: palette indexes, x fastest, then y, then z.
type GraphData struct {
	Name    string
	L, H, W int
	Voxels  []byte
}

type IndexData struct {
	Name string
	Data []byte
}

// CartoonData is one cartoon. Normal and Audio cartoons use Header and Elements, any other type
// (Script) is stored as the raw words of Script.
type CartoonData struct {
	Type     int // Normal, Script or Audio
	Header   FrameHeader
	Elements []ElementData
	Script   []uint32
}

// ElementData is one element; only the prop list matching Kind is used.
type ElementData struct {
	Kind   int // SetGraph, DrawBall or TitleStyle
	Graphs []GraphProp
	Balls  []BallProp
	Titles []TitleProp
}

type GraphProp struct {
	Start, End             int
	Graph, Palette         string
	X0, X1, Y0, Y1, Z0, Z1 int
	Angle0, Angle1         int
	Axis                   int // render.AxisX, AxisY or AxisZ
	Mode                   int // render.ModeSet or ModeMix
}

// BallProp is a sphere; centre and radius are in 1/100 voxel, colors are 0bRRGGBB.
type BallProp struct {
	Start, End        int
	X, Y, Z           int
	R0, R1            int
	Hollow            bool
	ColorIn, ColorOut byte
}

type TitleProp struct {
	Start, End            int
	Graph, Palette, Index string // font strip graph, palette, path index
	High                  int    // y position
	Style                 int    // 0 = forward, 1 = reverse
}

// FromBlob converts parsed show data into the editable model. Elements of unknown kinds are
// skipped (the player ignores them too).
func FromBlob(b *Blob) (*Show, error) {
	data := b.data
	show := &Show{}

	config := b.sec[secConfig]
	if config.len < 8 || 8+8*b.DisplayCount() > config.len {
		return nil, common.ErrBlobLayout
	}
	for key := range show.Config.Keys {
		show.Config.Keys[key] = byte(b.KeyFunction(key))
	}
	for i := 0; i < b.DisplayCount(); i++ {
		show.Config.Displays = append(show.Config.Displays, b.Display(i))
	}

	section := b.sec[secPalette]
	for offset := section.off; offset+32 <= section.off+section.len; offset += 32 {
		palette := Palette{Name: Name(data, offset)}
		copy(palette.Colors[:], data[offset+nameSize:offset+32])
		show.Palettes = append(show.Palettes, palette)
	}

	section = b.sec[secGraph]
	for offset, end := section.off, section.off+section.len; offset+28 <= end; {
		size, dataLen := int(U32(data, offset+16)), int(U32(data, offset+20))
		if size < 28 || offset+size > end || 28+dataLen > size {
			return nil, common.ErrBlobLayout
		}

		graph := b.graphAt(offset)
		voxels := make([]byte, graph.Voxels())
		if err := graph.Decode(voxels); err != nil {
			return nil, err
		}

		show.Graphs = append(show.Graphs, GraphData{
			Name: graph.Name, L: graph.Length, H: graph.Height, W: graph.Width, Voxels: voxels})
		offset += size
	}

	section = b.sec[secIndex]
	for offset, end := section.off, section.off+section.len; offset+20 <= end; {
		n := int(U32(data, offset+16))
		if offset+20+n > end {
			return nil, common.ErrBlobLayout
		}

		show.Indexes = append(show.Indexes, IndexData{Name: Name(data, offset), Data: []byte(data[offset+20 : offset+20+n])})
		offset += 20 + (n+3)&^3
	}

	for group := range show.Groups {
		for _, cartoon := range b.Group(group) {
			show.Groups[group] = append(show.Groups[group], cartoonData(cartoon))
		}
	}
	return show, nil
}

func cartoonData(cartoon Cartoon) CartoonData {
	result := CartoonData{Type: cartoon.Type}
	body := cartoon.Body
	if cartoon.Type != Normal && cartoon.Type != Audio {
		for offset := 0; offset+4 <= len(body); offset += 4 {
			result.Script = append(result.Script, U32(body, offset))
		}
		return result
	}

	result.Header = cartoon.Header()
	for element := 8; element+8 <= len(body); {
		size, kind := int(U32(body, element)), int(U32(body, element+4))
		if size < 8 || element+size > len(body) {
			break
		}

		if kind >= 0 && kind < len(propSizes) {
			item := ElementData{Kind: kind}
			for position := element + 8; position+propSizes[kind] <= element+size; position += propSizes[kind] {
				item.add(body, position)
			}
			result.Elements = append(result.Elements, item)
		}
		element += size
	}
	return result
}

// add reads the prop at position of a cartoon body.
func (this *ElementData) add(body string, position int) {
	start, end := I16(body, position), I16(body, position+2)
	data := body[position+4:]
	field := func(offset int) int { return I16(data, offset) }
	switch this.Kind {
	case SetGraph:
		this.Graphs = append(this.Graphs, GraphProp{
			Start: start, End: end,
			Graph: Name(data, 0), Palette: Name(data, 16),
			X0: field(32), X1: field(34), Y0: field(36), Y1: field(38), Z0: field(40), Z1: field(42),
			Angle0: field(44), Angle1: field(46),
			Axis: int(uint16(field(48))), Mode: int(uint16(field(50))),
		})
	case DrawBall:
		this.Balls = append(this.Balls, BallProp{
			Start: start, End: end,
			X: field(0), Y: field(2), Z: field(4), R0: field(6), R1: field(8),
			Hollow:  uint16(field(10)) != 0,
			ColorIn: data[12], ColorOut: data[13],
		})
	case TitleStyle:
		this.Titles = append(this.Titles, TitleProp{
			Start: start, End: end,
			Graph: Name(data, 0), Palette: Name(data, 16), Index: Name(data, 32),
			High: field(48), Style: int(uint16(field(50))),
		})
	}
}
