package lightshow_test

import (
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/lightshow"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/blob"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/keyframe"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/render"
	"github.com/stretchr/testify/assert"
)

// A keyframe cartoon and the same cartoon compiled to show data must render the same frames.
func TestKeyframeMatchesShowData(t *testing.T) {
	// Arrange
	colors := [16]byte{0, 0x03, 0x0C, 0x0F, 0x30, 0x33, 0x3C, 0x3F, 0x15, 0x2A}
	library := &keyframe.Library{
		Shapes:   []keyframe.Shape{{Name: "box", L: 2, H: 2, W: 2, Art: "12 34\n56 78"}},
		Palettes: []keyframe.Palette{{Name: "pal", Colors: string(colors[:])}},
	}
	cartoon := &keyframe.Cartoon{
		Frames: 12, Loop: 1, Delay: 40, Library: library,
		Elements: []keyframe.Element{
			{
				keyframe.Graph{Start: 0, End: 5, Shape: "box", Palette: "pal", To: keyframe.Pose{X: 6, Y: 6, Z: 6, Angle: 90}, Axis: render.AxisY, Mode: render.ModeMix},
				keyframe.Graph{Start: 2, End: 11, Shape: "box", From: keyframe.Pose{X: 3, Angle: -30}, To: keyframe.Pose{Y: 5, Angle: 200}, Axis: render.AxisX},
			},
			{keyframe.Ball{Start: 0, End: 11, X: 350, Y: 350, Z: 350, From: 100, To: 500, Hollow: true, Out: 0x30, In: 0x0C}},
		},
	}
	show := &blob.Show{
		Palettes: []blob.Palette{{Name: "pal", Colors: colors}},
		Graphs:   []blob.GraphData{{Name: "box", L: 2, H: 2, W: 2, Voxels: []byte{5, 6, 1, 2, 7, 8, 3, 4}}},
		Groups: [blob.GroupCount][]blob.CartoonData{{{
			Type:   blob.Normal,
			Header: blob.FrameHeader{FrameCount: 12, Loop: 1, Delay: 40},
			Elements: []blob.ElementData{
				{Kind: blob.SetGraph, Graphs: []blob.GraphProp{
					{Start: 0, End: 5, Graph: "box", Palette: "pal", X1: 6, Y1: 6, Z1: 6, Angle1: 90, Axis: render.AxisY, Mode: render.ModeMix},
					{Start: 2, End: 11, Graph: "box", X0: 3, Y1: 5, Angle0: -30, Angle1: 200, Axis: render.AxisX},
				}},
				{Kind: blob.DrawBall, Balls: []blob.BallProp{
					{Start: 0, End: 11, X: 350, Y: 350, Z: 350, R0: 100, R1: 500, Hollow: true, ColorOut: 0x30, ColorIn: 0x0C},
				}},
			},
		}}},
	}
	compiled := lightshow.NewBlobShow(compile(t, show), 0).Cartoon(0)
	library.Missing = func(kind, name string) { t.Errorf("missing %s %q", kind, name) }

	// Act
	// Assert
	assert.Equal(t, compiled.Kind(), cartoon.Kind())
	assert.Equal(t, compiled.Header(), cartoon.Header())
	for frame := 0; frame < 12; frame++ {
		want, got := newCanvas(), newCanvas()
		compiled.Render(frame, want)
		cartoon.Render(frame, got)
		assert.Equal(t, want.Displays[0].Voxels, got.Displays[0].Voxels, "frame %d", frame)
		assert.True(t, lit(got.Displays[0].Voxels), "frame %d is empty", frame)
	}
}
