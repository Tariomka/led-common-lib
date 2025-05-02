package led_test

import (
	"image/color"
	"math/rand"
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/led"
	"github.com/stretchr/testify/assert"
)

func TestToProcessorColor(t *testing.T) {
	t.Run("WhenAlphaIsZero", func(t *testing.T) {
		// Arrange
		// Act
		actual := led.RGBAToColor(color.RGBA{R: 0, G: 0, B: 0, A: 0})

		// Assert
		assert.Equal(t, led.NoColor, actual)
	})

	t.Run("WhenAlphaIsZeroAndColorsAreNotZero", func(t *testing.T) {
		// Arrange
		color := color.RGBA{
			R: uint8(rand.Intn(8)),
			G: uint8(rand.Intn(8)),
			B: uint8(rand.Intn(8)),
			A: 0,
		}

		// Act
		actual := led.RGBAToColor(color)

		// Assert
		assert.Equal(t, led.NoColor, actual)
	})

	t.Run("WhenAlphaIsNotZeroAndColorsAreZero", func(t *testing.T) {
		// Arrange
		color := color.RGBA{
			R: 0,
			G: 0,
			B: 0,
			A: 255,
		}

		// Act
		actual := led.RGBAToColor(color)

		// Assert
		assert.Equal(t, led.NoColor, actual)
	})

	t.Run("WhenRedIsNotZero", func(t *testing.T) {
		// Arrange
		// Act
		actual := led.RGBAToColor(color.RGBA{R: 255, G: 0, B: 0, A: 255})

		// Assert
		assert.Equal(t, led.Red, actual)
	})

	t.Run("WhenGreenIsNotZero", func(t *testing.T) {
		// Arrange
		// Act
		actual := led.RGBAToColor(color.RGBA{R: 0, G: 255, B: 0, A: 255})

		// Assert
		assert.Equal(t, led.Green, actual)
	})

	t.Run("WhenBlueIsNotZero", func(t *testing.T) {
		// Arrange
		// Act
		actual := led.RGBAToColor(color.RGBA{R: 0, G: 0, B: 255, A: 255})

		// Assert
		assert.Equal(t, led.Blue, actual)
	})

	t.Run("WhenRedAndGreenIsNotZero", func(t *testing.T) {
		// Arrange
		// Act
		actual := led.RGBAToColor(color.RGBA{R: 255, G: 255, B: 0, A: 255})

		// Assert
		assert.Equal(t, led.Yellow, actual)
	})

	t.Run("WhenRedAndBlueIsNotZero", func(t *testing.T) {
		// Arrange
		// Act
		actual := led.RGBAToColor(color.RGBA{R: 255, G: 0, B: 255, A: 255})

		// Assert
		assert.Equal(t, led.Violet, actual)
	})

	t.Run("WhenGreenAndBlueIsNotZero", func(t *testing.T) {
		// Arrange
		// Act
		actual := led.RGBAToColor(color.RGBA{R: 0, G: 255, B: 255, A: 255})

		// Assert
		assert.Equal(t, led.Cyan, actual)
	})

	t.Run("WhenRedAndGreenAndBlueIsNotZero", func(t *testing.T) {
		// Arrange
		// Act
		actual := led.RGBAToColor(color.RGBA{R: 255, G: 255, B: 255, A: 255})

		// Assert
		assert.Equal(t, led.White, actual)
	})
}
