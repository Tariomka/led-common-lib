package blob_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/common"
	"github.com/Tariomka/led-common-lib/pkg/lightshow"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/blob"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func randomVoxels(random *rand.Rand, count, values int) []byte {
	voxels := make([]byte, count)
	for i := range voxels {
		voxels[i] = byte(random.Intn(values))
	}
	return voxels
}

func sampleShow() *blob.Show {
	random := rand.New(rand.NewSource(1))
	return &blob.Show{
		Config: blob.Config{
			Keys:     [7]byte{2, 3, 4, 6, 8, 7, 9},
			Displays: []blob.Display{{CubeType: 1}, {X: 8, Y: -8, Z: 16, Dir: 1, CubeType: 2}},
		},
		Palettes: []blob.Palette{
			{Name: "red", Colors: [16]byte{0, 0x30, 0x30}},
			{Name: "fifteen-chars-x", Colors: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}},
		},
		Graphs: []blob.GraphData{
			{Name: "box", L: 2, H: 2, W: 2, Voxels: []byte{5, 6, 1, 2, 7, 8, 3, 4}},
			{Name: "random", L: 8, H: 8, W: 8, Voxels: randomVoxels(random, 512, 3)},
			{Name: "colorful", L: 4, H: 4, W: 4, Voxels: randomVoxels(random, 64, 20)},
			{Name: "empty", L: 8, H: 8, W: 8, Voxels: make([]byte, 512)},
		},
		Indexes: []blob.IndexData{{Name: "path", Data: []byte{0, 0, 0, 0, 1, 0, 0, 0, 2, 0}}},
		Groups: [blob.GroupCount][]blob.CartoonData{
			{
				{
					Type:   blob.Normal,
					Header: blob.FrameHeader{FrameCount: 20, Loop: 2, Delay: 50},
					Elements: []blob.ElementData{
						{Kind: blob.SetGraph, Graphs: []blob.GraphProp{
							{Start: 0, End: 9, Graph: "box", Palette: "red", X1: 6, Y1: 6, Z1: 6, Angle1: 90, Axis: render.AxisY, Mode: render.ModeMix},
							{Start: 5, End: 19, Graph: "random", X0: -2, X1: 2, Angle0: -45, Angle1: 45, Axis: render.AxisZ},
						}},
						{Kind: blob.DrawBall, Balls: []blob.BallProp{
							{Start: 0, End: 19, X: 350, Y: 350, Z: 350, R0: 100, R1: 400, Hollow: true, ColorIn: 0x03, ColorOut: 0x30},
						}},
					},
				},
				{
					Type:   blob.Audio,
					Header: blob.FrameHeader{FrameCount: 10, Loop: 2, Delay: 10},
					Elements: []blob.ElementData{
						{Kind: blob.TitleStyle, Titles: []blob.TitleProp{
							{Start: 0, End: 63, Graph: "colorful", Palette: "fifteen-chars-x", Index: "path", High: 2, Style: 1},
						}},
					},
				},
			},
			nil,
			{{Type: blob.Script, Script: []uint32{2, 0, 28 << 16}}},
		},
	}
}

func TestMarshal(t *testing.T) {
	t.Run("WhenParsedBack", func(t *testing.T) {
		// Arrange
		show := sampleShow()

		// Act
		data, err := show.Marshal()
		require.NoError(t, err)
		parsed, err := blob.Parse(string(data))
		require.NoError(t, err)
		actual, err := blob.FromBlob(parsed)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, show, actual)
		assert.Equal(t, len(data), parsed.Size())
	})

	t.Run("WhenGraphsAreCompressed", func(t *testing.T) {
		// Arrange
		show := sampleShow()
		long := make([]byte, 255*8*16)
		long[len(long)-1] = 1
		show.Graphs = append(show.Graphs, blob.GraphData{Name: "long", L: 255, H: 8, W: 16, Voxels: long})
		expected := map[string]int{
			"box": blob.NoCompress, "random": blob.Normal4, "colorful": blob.NoCompress, "empty": blob.Continue2, "long": blob.Continue8,
		}

		// Act
		data, err := show.Marshal()
		require.NoError(t, err)
		parsed, err := blob.Parse(string(data))
		require.NoError(t, err)

		// Assert
		for _, source := range show.Graphs {
			graph, ok := parsed.Graph(source.Name)
			require.True(t, ok, source.Name)
			assert.Equal(t, expected[source.Name], graph.Compression, source.Name)

			voxels := make([]byte, graph.Voxels())
			require.NoError(t, graph.Decode(voxels))
			assert.Equal(t, source.Voxels, voxels, source.Name)
		}
	})

	t.Run("WhenNameIsTooLong", func(t *testing.T) {
		// Arrange
		show := sampleShow()
		show.Palettes[0].Name = "sixteen-chars-xx"

		// Act
		_, err := show.Marshal()

		// Assert
		assert.ErrorIs(t, err, common.ErrBlobNameTooLong)
	})

	t.Run("WhenDataIsTooLarge", func(t *testing.T) {
		// Arrange
		show := sampleShow()
		random := rand.New(rand.NewSource(2))
		show.Graphs = append(show.Graphs, blob.GraphData{Name: "huge", L: 255, H: 255, W: 3, Voxels: randomVoxels(random, 255*255*3, 32)})

		// Act
		_, err := show.Marshal()

		// Assert
		assert.ErrorIs(t, err, common.ErrBlobTooLarge)
	})

	t.Run("WhenValueIsOutOfRange", func(t *testing.T) {
		// Arrange
		show := sampleShow()
		show.Config.Displays[0].X = 40000

		// Act
		_, err := show.Marshal()

		// Assert
		assert.ErrorIs(t, err, common.ErrBlobValue)
	})

	t.Run("WhenGraphSizeDoesNotMatchVoxels", func(t *testing.T) {
		// Arrange
		show := sampleShow()
		show.Graphs[0].W = 3

		// Act
		_, err := show.Marshal()

		// Assert
		assert.ErrorIs(t, err, common.ErrBlobValue)
	})
}

// The factory show data, imported and compiled again, must play exactly like the original.
func TestMarshalFactoryShow(t *testing.T) {
	// Arrange
	original := parseOriginal(t)
	show, err := blob.FromBlob(original)
	require.NoError(t, err)

	// Act
	data, err := show.Marshal()
	require.NoError(t, err)
	compiled, err := blob.Parse(string(data))
	require.NoError(t, err)

	// Assert
	again, err := blob.FromBlob(compiled)
	require.NoError(t, err)
	assert.Equal(t, show, again)

	for group := 0; group < blob.GroupCount; group++ {
		want, got := lightshow.NewBlobShow(original, group), lightshow.NewBlobShow(compiled, group)
		require.Equal(t, want.Count(), got.Count(), "group %d", group+1)
		for key := 0; key < 7; key++ {
			assert.Equal(t, want.KeyFunction(key), got.KeyFunction(key))
		}

		for i := 0; i < want.Count(); i++ {
			wantCartoon, gotCartoon := want.Cartoon(i), got.Cartoon(i)
			require.Equal(t, wantCartoon.Kind(), gotCartoon.Kind())
			require.Equal(t, wantCartoon.Header(), gotCartoon.Header())

			frames := wantCartoon.Header().FrameCount
			if wantCartoon.Kind() == blob.Audio {
				frames = 64
			}
			wantCanvas, gotCanvas := newCanvas(), newCanvas()
			for frame := 0; frame < frames; frame++ {
				wantCanvas.Clear(0)
				gotCanvas.Clear(0)
				wantCartoon.Render(frame, wantCanvas)
				gotCartoon.Render(frame, gotCanvas)
				if !bytes.Equal(wantCanvas.Displays[0].Voxels, gotCanvas.Displays[0].Voxels) {
					t.Errorf("group %d cartoon %d frame %d differs", group+1, i, frame)
					break
				}
			}
		}
	}
	t.Logf("factory show data: %d bytes, compiled again: %d bytes", original.Size(), len(data))
}

func newCanvas() *render.Canvas {
	return &render.Canvas{Displays: []render.Display{render.NewDisplay(0, 0, 0, 8, 8, 8)}}
}
