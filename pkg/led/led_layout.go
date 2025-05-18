package led

import (
	"iter"

	"github.com/Tariomka/led-common-lib/pkg/common"
)

const (
	layerCount        = uint8(8)
	colorCount        = uint8(3)
	baseLedCount      = uint8(8) // The count of leds in a row/column. Row and Column count is the same
	bytesInLayerCount = colorCount * baseLedCount
)

// LedLayout is a state representation of all led colors of the cube.
//
// There are 8 layers, each having 64 leds, each having 3 contacts (GBR),
// making a single layer hold 192 bits of information.
//
// 12 shift registers control all 192 activations, each shift register having 16 activations
// and all shift registers connected in series.
// Shift registers receive data in little-endian (Least significant bit first)
//
// First 8 bytes [0-7] control the Green color, next 8 [8-15] - Blue, last 8 [16-23] - Red
// First byte in a color block controlls the back side of the layer, last (8'th) byte - the front.
// Bits from right to left control each led from right to left led,
// i.e. 0b00000001 turns on the right most led, 0b10000000 - left most led.
type LedLayout [layerCount][bytesInLayerCount]byte

func (this *LedLayout) IterateSlices() iter.Seq2[uint8, []byte] {
	return func(yield func(uint8, []byte) bool) {
		for zAxis := range layerCount {
			if !yield(zAxis, this[zAxis][:]) {
				return
			}
		}
	}
}

func (this *LedLayout) IterateColors() iter.Seq2[Index, Color] {
	return func(yield func(Index, Color) bool) {
		for zAxis := range layerCount {
			for yAxis := range baseLedCount {
				for xAxis := range baseLedCount {
					if !yield(
						Index{X: xAxis, Y: yAxis, Z: zAxis},
						this.getColor(xAxis, yAxis, zAxis)) {
						return
					}
				}
			}
		}
	}
}

func (this *LedLayout) Overwrite(iterator iter.Seq2[uint8, []byte]) error {
	for index, layer := range iterator {
		if index >= baseLedCount {
			return nil
		}

		if len(layer) < int(bytesInLayerCount) {
			return common.ErrNotEnoughData
		}

		this[index] = [bytesInLayerCount]byte(layer)
	}
	return nil
}

func (this *LedLayout) ChangeSingle(x, y, z uint8, c Color) error {
	if err := validateAxes(x, y, z); err != nil {
		return err
	}

	this.resetBit(x, y, z)
	if c != NoColor {
		this.setBit(x, y, z, c)
	}
	return nil
}

func (this *LedLayout) SetSingle(x, y, z uint8, c Color) error {
	if err := validateAxes(x, y, z); err != nil {
		return err
	}

	if c == NoColor {
		this.resetBit(x, y, z)
		return nil
	}

	this.setBit(x, y, z, c)
	return nil
}

func (this *LedLayout) ResetSingle(x, y, z uint8) error {
	if err := validateAxes(x, y, z); err != nil {
		return err
	}

	this.resetBit(x, y, z)
	return nil
}

func (this *LedLayout) ChangeRowIndividual(y, z uint8, c Color, rowValues byte) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	this.resetByte(y, z, rowValues)
	if c != NoColor {
		this.setByte(y, z, c, rowValues)
	}
	return nil
}

func (this *LedLayout) SetRowIndividual(y, z uint8, c Color, rowValues byte) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	if c == NoColor {
		this.resetByte(y, z, rowValues)
		return nil
	}

	this.setByte(y, z, c, rowValues)
	return nil
}

func (this *LedLayout) ResetRowIndividual(y, z uint8, rowValues byte) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	this.resetByte(y, z, rowValues)
	return nil
}

func (this *LedLayout) ChangeRow(y, z uint8, c Color) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	this.resetByte(y, z, all)
	if c != NoColor {
		this.setByte(y, z, c, all)
	}
	return nil
}

func (this *LedLayout) SetRow(y, z uint8, c Color) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	if c == NoColor {
		this.resetByte(y, z, all)
		return nil
	}

	this.setByte(y, z, c, all)
	return nil
}

func (this *LedLayout) ResetRow(y, z uint8) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	this.resetByte(y, z, all)
	return nil
}

func (this *LedLayout) ChangeLayer(z uint8, c Color) error {
	if err := validateLayer(z); err != nil {
		return err
	}

	this.resetBytes(z)
	if c != NoColor {
		this.setBytes(z, c)
	}
	return nil
}

func (this *LedLayout) SetLayer(z uint8, c Color) error {
	if err := validateLayer(z); err != nil {
		return err
	}

	if c == NoColor {
		this.resetBytes(z)
		return nil
	}

	this.setBytes(z, c)
	return nil
}

func (this *LedLayout) ResetLayer(z uint8) error {
	if err := validateLayer(z); err != nil {
		return err
	}

	this.resetBytes(z)
	return nil
}

func (this *LedLayout) ChangeBlock(c Color) {
	this.resetAll()
	if c != NoColor {
		this.setAll(c)
	}
}

func (this *LedLayout) SetBlock(c Color) {
	if c == NoColor {
		this.resetAll()
		return
	}

	this.setAll(c)
}

func (this *LedLayout) ResetBlock() {
	this.resetAll()
}

func (this *LedLayout) setBit(x, y, z uint8, c Color) {
	for _, index := range layoutOffsetIndex(y, c) {
		this.mutateByteWithOr(index, z, 1<<x)
	}
}

func (this *LedLayout) resetBit(x, y, z uint8) {
	for _, index := range layoutOffsetIndex(y, White) {
		this.mutateByteWithAnd(index, z, ^(1 << x))
	}
}

func (this *LedLayout) setByte(y, z uint8, c Color, value byte) {
	for _, index := range layoutOffsetIndex(y, c) {
		this.mutateByteWithOr(index, z, value)
	}
}

func (this *LedLayout) resetByte(y, z uint8, value byte) {
	for _, index := range layoutOffsetIndex(y, White) {
		this.mutateByteWithAnd(index, z, ^value)
	}
}

func (this *LedLayout) setBytes(z uint8, c Color) {
	for _, offset := range layoutOffsetIndex(0, c) {
		for index := offset; index < offset+baseLedCount; index++ {
			this.mutateByteWithOr(index, z, all)
		}
	}
}

func (this *LedLayout) resetBytes(z uint8) {
	for index := range bytesInLayerCount {
		this.mutateByteWithAnd(index, z, none)
	}
}

func (this *LedLayout) setAll(c Color) {
	for layer := range layerCount {
		for _, offset := range layoutOffsetIndex(0, c) {
			for index := offset; index < offset+baseLedCount; index++ {
				this.mutateByteWithOr(index, layer, all)
			}
		}
	}
}

func (this *LedLayout) resetAll() {
	for layer := range layerCount {
		for index := range bytesInLayerCount {
			this.mutateByteWithAnd(index, layer, none)
		}
	}
}

func (this *LedLayout) mutateByteWithAnd(index, layer, value uint8) {
	this[layer][index] &= value
}

func (this *LedLayout) mutateByteWithOr(index, layer, value uint8) {
	this[layer][index] |= value
}

func (this *LedLayout) getColor(x, y, z uint8) Color {
	var color Color

	for shift := range colorCount {
		if this[z][shift*baseLedCount+y]>>x&1 == 1 {
			color |= 1 << shift
		}
	}

	return color
}

func layoutOffsetIndex(index uint8, c Color) []uint8 {
	offsets := []uint8{}

	for shift := range colorCount {
		if c>>shift&1 == 1 {
			offsets = append(offsets, shift*baseLedCount+index)
		}
	}

	return offsets
}

func validateAxes(x, y, z uint8) error {
	if x > 7 || y > 7 || z > 7 {
		return common.ErrOutOfBounds
	}
	return nil
}

func validateRow(y, z uint8) error {
	if y > 7 || z > 7 {
		return common.ErrOutOfBounds
	}
	return nil
}

func validateLayer(z uint8) error {
	if z > 7 {
		return common.ErrOutOfBounds
	}
	return nil
}
