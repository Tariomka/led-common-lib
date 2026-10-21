// Package keyframe describes light shows in Go: a cartoon is a list of elements, each a list of
// keyframed properties (graph moves/rotations, balls) interpolated between their start and end
// frames, drawn with the shapes and palettes of a Library. It is the Go form of the original
// iCubeSmart show data and renders the same frames.
package keyframe

import (
	"github.com/Tariomka/led-common-lib/pkg/lightshow"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/blob"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/render"
)

// Pose is a graph position (minimum corner, voxels) and rotation angle (degrees).
type Pose struct{ X, Y, Z, Angle int }

// Graph draws a shape, moving and rotating it from From to To between frames Start and End.
// Shape and Palette are library names and may end in a frame expression ("Fire$C%4", see
// blob.ResolveName); an empty Palette is render.DefaultPalette.
type Graph struct {
	Start, End     int
	Shape, Palette string
	From, To       Pose
	Axis           int // render.AxisX, AxisY or AxisZ
	Mode           int // render.ModeSet or ModeMix
}

// Ball draws a sphere whose radius goes from From to To. Centre and radius are in 1/100 voxel.
type Ball struct {
	Start, End int
	X, Y, Z    int
	From, To   int
	Hollow     bool
	Out, In    byte // colors of the outer shell and of the inside
}

// Prop is a Graph or a Ball.
type Prop interface {
	span() (start, end int)
	draw(frame int, library *Library, canvas *render.Canvas) int
}

// Element groups props; like the original engine, props after an interpolating prop of the same
// element see the frame shifted by that prop's start.
type Element []Prop

type Cartoon struct {
	Audio    bool // frame = microphone level 0..63, see Frames and Loop
	Frames   int  // frame count; Audio: duration in seconds, 0 = until changed
	Loop     int  // repeat count; Audio: level mode
	Delay    int  // ms per frame at normal speed
	Elements []Element
	Library  *Library
}

func (this *Cartoon) Kind() int {
	if this.Audio {
		return blob.Audio
	}
	return blob.Normal
}

func (this *Cartoon) Header() blob.FrameHeader {
	return blob.FrameHeader{FrameCount: this.Frames, Loop: this.Loop, Delay: this.Delay}
}

func (this *Cartoon) Render(frame int, canvas *render.Canvas) {
	for _, element := range this.Elements {
		inner := frame
		for _, prop := range element {
			if start, end := prop.span(); inner >= start && inner <= end {
				inner = prop.draw(inner, this.Library, canvas)
			}
		}
	}
}

func lerp(v0, v1, frame, start, end int) int {
	return v0 + (v1-v0)*(frame-start)/(end-start)
}

func (this Graph) span() (int, int) { return this.Start, this.End }

func (this Graph) draw(frame int, library *Library, canvas *render.Canvas) int {
	pose := this.From
	switch frame {
	case this.Start:
	case this.End:
		pose = this.To
	default:
		pose = Pose{
			X:     lerp(this.From.X, this.To.X, frame, this.Start, this.End),
			Y:     lerp(this.From.Y, this.To.Y, frame, this.Start, this.End),
			Z:     lerp(this.From.Z, this.To.Z, frame, this.Start, this.End),
			Angle: lerp(this.From.Angle, this.To.Angle, frame, this.Start, this.End),
		}
		frame -= this.Start
	}
	shape, ok := library.shape(blob.ResolveName(this.Shape, frame, this.Start))
	if !ok {
		return frame
	}
	palette, ok := library.palette(blob.ResolveName(this.Palette, frame, this.Start))
	if !ok {
		return frame
	}
	canvas.Graph(shape, palette, pose.X, pose.Y, pose.Z, this.Axis, pose.Angle, this.Mode)
	return frame
}

func (this Ball) span() (int, int) { return this.Start, this.End }

func (this Ball) draw(frame int, _ *Library, canvas *render.Canvas) int {
	radius := this.From
	switch frame {
	case this.Start:
	case this.End:
		radius = this.To
	default:
		radius = lerp(this.From, this.To, frame, this.Start, this.End)
		frame -= this.Start
	}
	canvas.Ball(this.X, this.Y, this.Z, radius, this.Out, this.In, this.Hollow)
	return frame
}

// Show is a list of cartoons built on demand, with the key functions of KEY1..KEY7.
type Show struct {
	Cartoons []func() *Cartoon
	Keys     [7]int // lightshow.Action* codes
	Library  *Library
}

func (this *Show) Count() int { return len(this.Cartoons) }

func (this *Show) Cartoon(index int) lightshow.Cartoon { return this.Cartoons[index]() }

func (this *Show) SetMissing(report func(kind, name string)) {
	if this.Library != nil {
		this.Library.Missing = report
	}
}

func (this *Show) KeyFunction(key int) int {
	if key < 0 || key >= len(this.Keys) {
		return 0
	}
	return this.Keys[key]
}
