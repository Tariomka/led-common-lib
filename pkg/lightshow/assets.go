package lightshow

import (
	"github.com/Tariomka/led-common-lib/pkg/lightshow/blob"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/render"
)

// assets resolves resource names and keeps the most recently decoded graph.
type assets struct {
	blob     *blob.Blob
	buf      []byte
	cachedID int
	cached   render.Voxels
	Missing  func(kind, name string) // optional, for diagnostics
}

func newAssets(b *blob.Blob) *assets {
	return &assets{blob: b, cachedID: -1}
}

func (this *assets) graph(name string) (render.Voxels, bool) {
	graph, ok := this.blob.Graph(name)
	if !ok {
		this.missing("graph", name)
		return render.Voxels{}, false
	}

	if graph.ID() == this.cachedID {
		return this.cached, true
	}

	voxels := graph.Voxels()
	if cap(this.buf) < voxels {
		this.buf = make([]byte, voxels)
	}
	if err := graph.Decode(this.buf[:voxels]); err != nil {
		this.missing("graph data", name)
		return render.Voxels{}, false
	}

	this.cachedID = graph.ID()
	this.cached = render.Voxels{L: graph.Length, H: graph.Height, W: graph.Width, Data: this.buf[:voxels]}
	return this.cached, true
}

// palette returns the default palette for an empty name.
func (this *assets) palette(name string) (string, bool) {
	if name == "" {
		return render.DefaultPalette, true
	}

	palette, ok := this.blob.Palette(name)
	if !ok {
		this.missing("palette", name)
	}
	return palette, ok
}

func (this *assets) index(name string) (string, bool) {
	index, ok := this.blob.Index(name)
	if !ok {
		this.missing("index", name)
	}
	return index, ok
}

func (this *assets) missing(kind, name string) {
	if this.Missing != nil {
		this.Missing(kind, name)
	}
}

func linearInterpolation(v0, v1, frame, start, end int) int {
	return v0 + (v1-v0)*(frame-start)/(end-start)
}

// renderFrame draws all element properties of a Normal/Audio cartoon that are active at frame.
func renderFrame(cartoon blob.Cartoon, frame int, canvas *render.Canvas, assetsObject *assets) {
	elementStart := -1
	innerIndex := frame
	cartoon.ForEachProp(func(prop blob.Prop) {
		// The vendor engine reuses the frame variable across the properties of one element and
		// subtracts the property start from it while interpolating. Reproduced as-is.
		if prop.Element != elementStart {
			elementStart, innerIndex = prop.Element, frame
		}
		if innerIndex < prop.Start || innerIndex > prop.End {
			return
		}

		switch prop.Kind {
		case blob.SetGraph:
			innerIndex = drawGraph(prop, innerIndex, canvas, assetsObject)
		case blob.DrawBall:
			innerIndex = drawBall(prop, innerIndex, canvas)
		case blob.TitleStyle:
			innerIndex = drawTitle(prop, innerIndex, canvas, assetsObject)
		}
	})
}

func drawGraph(prop blob.Prop, frame int, canvas *render.Canvas, a *assets) int {
	data := prop.Data
	field := func(i int) int { return blob.I16(data, 32+2*i) }
	var x, y, z, angle int
	switch frame {
	case prop.Start:
		x, y, z, angle = field(0), field(2), field(4), field(6)
	case prop.End:
		x, y, z, angle = field(1), field(3), field(5), field(7)
	default:
		x = linearInterpolation(field(0), field(1), frame, prop.Start, prop.End)
		y = linearInterpolation(field(2), field(3), frame, prop.Start, prop.End)
		z = linearInterpolation(field(4), field(5), frame, prop.Start, prop.End)
		angle = linearInterpolation(field(6), field(7), frame, prop.Start, prop.End)
		frame -= prop.Start
	}
	axis, mode := int(uint16(field(8))), int(uint16(field(9)))
	graphName := blob.ResolveName(blob.Name(data, 0), frame, prop.Start)
	paletteName := blob.ResolveName(blob.Name(data, 16), frame, prop.Start)
	g, ok := a.graph(graphName)
	if !ok {
		return frame
	}

	pal, ok := a.palette(paletteName)
	if !ok {
		return frame
	}

	canvas.Graph(g, pal, x, y, z, axis, angle, mode)
	return frame
}

func drawBall(prop blob.Prop, frame int, canvas *render.Canvas) int {
	data := prop.Data
	r0, r1 := blob.I16(data, 6), blob.I16(data, 8)
	var r int
	switch frame {
	case prop.Start:
		r = r0
	case prop.End:
		r = r1
	default:
		r = linearInterpolation(r0, r1, frame, prop.Start, prop.End)
		frame -= prop.Start
	}
	hollow := uint16(blob.I16(data, 10)) != 0
	canvas.Ball(blob.I16(data, 0), blob.I16(data, 2), blob.I16(data, 4), r, data[13], data[12], hollow)
	return frame
}

func drawTitle(prop blob.Prop, frame int, canvas *render.Canvas, assetsObject *assets) int {
	data := prop.Data
	frame -= prop.Start
	graphName := blob.ResolveName(blob.Name(data, 0), frame, prop.Start)
	paletteName := blob.ResolveName(blob.Name(data, 16), frame, prop.Start)
	indexName := blob.ResolveName(blob.Name(data, 32), frame, prop.Start)
	g, ok := assetsObject.graph(graphName)
	if !ok {
		return frame
	}

	path, ok := assetsObject.index(indexName)
	if !ok {
		return frame
	}

	pal, ok := assetsObject.palette(paletteName)
	if !ok {
		return frame
	}

	canvas.Title(g, pal, path, blob.I16(data, 48), int(uint16(blob.I16(data, 50))), frame)
	return frame
}
