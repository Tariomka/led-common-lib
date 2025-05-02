package led

import "image/color"

type Color uint8

const (
	NoColor Color = 0b0
	Green   Color = 0b1
	Blue    Color = 0b10
	Red     Color = 0b100
	Cyan    Color = 0b11
	Yellow  Color = 0b101
	Violet  Color = 0b110
	White   Color = 0b111
)

const (
	none byte = 0b00000000
	all  byte = 0b11111111
)

func RGBAToColor(c color.RGBA) Color {
	if c.A < 1 {
		return NoColor
	}

	var color Color

	if c.G > 0 {
		color |= 1 << 0
	}
	if c.B > 0 {
		color |= 1 << 1
	}
	if c.R > 0 {
		color |= 1 << 2
	}

	return color
}
