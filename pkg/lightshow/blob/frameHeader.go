package blob

// Frame element kinds
const (
	SetGraph   = 0
	DrawBall   = 1
	TitleStyle = 2
)

// Frame header of Normal and Audio cartoons.
type FrameHeader struct {
	FrameCount int // Normal: frames; Audio: duration in seconds (0 = forever)
	Loop       int // Normal: repeat count; Audio: audio type
	Delay      int // ms per frame before speed scaling
}
