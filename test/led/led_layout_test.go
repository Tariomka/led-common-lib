package led_test

import (
	"math/rand"
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/common"
	"github.com/Tariomka/led-common-lib/pkg/led"
	"github.com/Tariomka/led-common-lib/test/helpers"
	"github.com/stretchr/testify/assert"
)

func TestLedLayout(t *testing.T) {
	// Arrange
	ledLayout := &led.LedLayout{}
	expected := [][]byte{
		{0x0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x0, 0x0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x0, 0x0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x0},
		{0xff, 0xfd, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		{0xff, 0xff, 0xfb, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		{0xff, 0xff, 0xff, 0xf7, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		{0xff, 0xff, 0xff, 0xff, 0xef, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xdf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xbf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xbf, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xbf, 0xff},
		{0x0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x0, 0x0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x0, 0x0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x80},
	}

	// Act
	ledLayout.ResetBlock()
	ledLayout.SetBlock(led.White)
	ledLayout.ResetRow(0, 0)
	ledLayout.ResetRow(7, 0)
	ledLayout.ResetRow(0, 7)
	ledLayout.ResetRow(7, 7)
	for i := uint8(1); i < 7; i++ {
		ledLayout.ChangeSingle(i, i, i, led.NoColor)
	}
	for i := uint8(2); i < 6; i++ {
		ledLayout.SetRowIndividual(i, i, led.Violet, 0b00111100)
	}
	ledLayout.SetSingle(7, 7, 7, led.Red)

	// Assert
	for index, value := range ledLayout.IterateSlices() {
		assert.Equal(t, expected[index], value)
	}
}

func TestLedLayout_ChangeSingle(t *testing.T) {
	t.Run("WhenInBoundsAndNoColor", func(t *testing.T) {
		// Arrange
		x := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := ^byte(1 << x)
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		err := ledLayout.ChangeSingle(x, y, z, led.NoColor)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)
	})

	t.Run("WhenInBoundsAndSingleColor", func(t *testing.T) {
		x := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(1 << x)
		testArgs := [][]any{
			{x, y, z, y, led.Green, expected},
			{x, y, z, y + 8, led.Blue, expected},
			{x, y, z, y + 16, led.Red, expected},
		}

		testCase := func(t *testing.T, x, y, z, index uint8, c led.Color, expected byte) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetSingle(x, y, z, led.White)

			// Act
			err := ledLayout.ChangeSingle(x, y, z, c)

			// Assert
			assert.Equal(t, expected, ledLayout[z][index])
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndTwoColors", func(t *testing.T) {
		x := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(1 << x)
		testArgs := [][]any{
			{x, y, z, []uint8{y, y + 8}, led.Cyan, expected},
			{x, y, z, []uint8{y, y + 16}, led.Yellow, expected},
			{x, y, z, []uint8{y + 8, y + 16}, led.Violet, expected},
		}

		testCase := func(t *testing.T, x, y, z uint8, indexes []uint8, c led.Color, expected byte) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetSingle(x, y, z, led.White)

			// Act
			err := ledLayout.ChangeSingle(x, y, z, c)

			// Assert
			for _, index := range indexes {
				assert.Equal(t, expected, ledLayout[z][index])
			}
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndThreeColors", func(t *testing.T) {
		// Arrange
		x := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(1 << x)
		ledLayout := &led.LedLayout{}
		ledLayout.SetSingle(x, y, z, led.White)

		// Act
		err := ledLayout.ChangeSingle(x, y, z, led.White)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)
	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		testArgs := [][]any{
			{uint8(100), uint8(1), uint8(2)},
			{uint8(3), uint8(200), uint8(4)},
			{uint8(5), uint8(6), uint8(69)},
			{uint8(0), uint8(70), uint8(69)},
			{uint8(101), uint8(5), uint8(69)},
			{uint8(99), uint8(10), uint8(3)},
			{uint8(8), uint8(9), uint8(10)},
		}

		testCase := func(t *testing.T, x, y, z uint8) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.ChangeSingle(x, y, z, led.Color(uint8(rand.Intn(8))))

			// Assert
			assert.ErrorIs(t, err, common.ErrOutOfBounds)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndColorIsNotPredefined", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		actual := ledLayout.ChangeSingle(0, 0, 0, led.Color(uint8(8+rand.Intn(247))))

		// Assert
		assert.Nil(t, actual)
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		xFirst := uint8(rand.Intn(8))
		xSecond := uint8(rand.Intn(8))
		xThird := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expectedGreen := ^byte(1<<xFirst) & ^byte(1<<xThird)
		expectedBlue := ^byte(1 << xFirst)
		expectedRed := ^byte(1<<xFirst) & ^byte(1<<xSecond) & ^byte(1<<xThird)
		ledLayout := &led.LedLayout{}
		ledLayout.SetRow(y, z, led.White)

		// Act
		ledLayout.ChangeSingle(xFirst, y, z, led.NoColor)
		ledLayout.ChangeSingle(xSecond, y, z, led.Cyan)
		ledLayout.ChangeSingle(xThird, y, z, led.Blue)

		// Assert
		assert.Equal(t, expectedGreen, ledLayout[z][y])
		assert.Equal(t, expectedBlue, ledLayout[z][y+8])
		assert.Equal(t, expectedRed, ledLayout[z][y+16])
	})
}

func TestLedLayout_ChangeRowIndividual(t *testing.T) {
	t.Run("WhenInBoundsAndNoColor", func(t *testing.T) {
		// Arrange
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(rand.Int())
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		err := ledLayout.ChangeRowIndividual(y, z, led.NoColor, ^expected)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)
	})

	t.Run("WhenInBoundsAndSingleColor", func(t *testing.T) {
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(rand.Int())
		testArgs := [][]any{
			{y, z, []uint8{y + 8, y + 16}, led.Green, expected},
			{y, z, []uint8{y, y + 16}, led.Blue, expected},
			{y, z, []uint8{y, y + 8}, led.Red, expected},
		}

		testCase := func(t *testing.T, y, z uint8, indexes []uint8, c led.Color, expected byte) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetBlock(led.White)

			// Act
			err := ledLayout.ChangeRowIndividual(y, z, c, ^expected)

			// Assert
			for _, index := range indexes {
				assert.Equal(t, expected, ledLayout[z][index])
			}
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndTwoColors", func(t *testing.T) {
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(rand.Int())
		testArgs := [][]any{
			{y, z, y + 16, led.Cyan, expected},
			{y, z, y + 8, led.Yellow, expected},
			{y, z, y, led.Violet, expected},
		}

		testCase := func(t *testing.T, y, z, index uint8, c led.Color, expected byte) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetBlock(led.White)

			// Act
			err := ledLayout.ChangeRowIndividual(y, z, c, ^expected)

			// Assert
			assert.Equal(t, expected, ledLayout[z][index])
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndThreeColors", func(t *testing.T) {
		// Arrange
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(rand.Int())
		ledLayout := &led.LedLayout{}

		// Act
		err := ledLayout.ChangeRowIndividual(y, z, led.White, expected)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)
	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		testArgs := [][]any{
			{uint8(100), uint8(1)},
			{uint8(2), uint8(200)},
			{uint8(68), uint8(69)},
		}

		testCase := func(t *testing.T, y, z uint8) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.ChangeRowIndividual(y, z, led.Color(uint8(rand.Intn(8))), byte(rand.Int()))

			// Assert
			assert.ErrorIs(t, err, common.ErrOutOfBounds)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndColorIsNotPredefined", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		err := ledLayout.ChangeRowIndividual(0, 0, led.Color(uint8(8+rand.Intn(247))), byte(rand.Int()))

		// Assert
		assert.Nil(t, err)
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		valueFirst := byte(1 << rand.Intn(8))
		valueSecond := byte(1 << rand.Intn(8))
		valueThird := byte(1 << rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expectedGreen := ^valueFirst & ^valueThird
		expectedBlue := ^valueFirst
		expectedRed := ^valueFirst & ^valueSecond & ^valueThird
		ledLayout := &led.LedLayout{}
		ledLayout.SetRow(y, z, led.White)

		// Act
		ledLayout.ChangeRowIndividual(y, z, led.NoColor, valueFirst)
		ledLayout.ChangeRowIndividual(y, z, led.Cyan, valueSecond)
		ledLayout.ChangeRowIndividual(y, z, led.Blue, valueThird)

		// Assert
		assert.Equal(t, expectedGreen, ledLayout[z][y])
		assert.Equal(t, expectedBlue, ledLayout[z][y+8])
		assert.Equal(t, expectedRed, ledLayout[z][y+16])
	})
}

func TestLedLayout_ChangeRow(t *testing.T) {
	t.Run("WhenInBoundsAndNoColor", func(t *testing.T) {
		// Arrange
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(0)
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		err := ledLayout.ChangeRow(y, z, led.NoColor)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)
	})

	t.Run("WhenInBoundsAndSingleColor", func(t *testing.T) {
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		testArgs := [][]any{
			{y, z, []uint8{y + 8, y + 16}, led.Green},
			{y, z, []uint8{y, y + 16}, led.Blue},
			{y, z, []uint8{y, y + 8}, led.Red},
		}

		testCase := func(t *testing.T, y, z uint8, indexes []uint8, c led.Color) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetBlock(led.White)

			// Act
			err := ledLayout.ChangeRow(y, z, c)

			// Assert
			for _, index := range indexes {
				assert.Equal(t, byte(0), ledLayout[z][index])
			}
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndTwoColors", func(t *testing.T) {
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		testArgs := [][]any{
			{y, z, y + 16, led.Cyan},
			{y, z, y + 8, led.Yellow},
			{y, z, y, led.Violet},
		}

		testCase := func(t *testing.T, y, z, index uint8, c led.Color) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetBlock(led.White)

			// Act
			err := ledLayout.ChangeRow(y, z, c)

			// Assert
			assert.Equal(t, byte(0), ledLayout[z][index])
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndThreeColors", func(t *testing.T) {
		// Arrange
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(255)
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.Red)

		// Act
		err := ledLayout.ChangeRow(y, z, led.White)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)
	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		testArgs := [][]any{
			{uint8(100), uint8(1)},
			{uint8(2), uint8(200)},
			{uint8(68), uint8(69)},
		}

		testCase := func(t *testing.T, y, z uint8) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.ChangeRow(y, z, led.Color(uint8(rand.Intn(8))))

			// Assert
			assert.ErrorIs(t, err, common.ErrOutOfBounds)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndColorIsNotPredefined", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		actual := ledLayout.ChangeRow(0, 0, led.Color(uint8(8+rand.Intn(247))))

		// Assert
		assert.Nil(t, actual)
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		ledLayout.ChangeRow(y, z, led.NoColor)
		ledLayout.ChangeRow(y, z, led.Cyan)
		ledLayout.ChangeRow(y, z, led.Blue)

		// Assert
		assert.Equal(t, byte(0), ledLayout[z][y])
		assert.Equal(t, byte(255), ledLayout[z][y+8])
		assert.Equal(t, byte(0), ledLayout[z][y+16])
	})
}

func TestLedLayout_ChangeLayer(t *testing.T) {
	t.Run("WhenInBoundsAndNoColor", func(t *testing.T) {
		// Arrange
		z := uint8(rand.Intn(8))
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		err := ledLayout.ChangeLayer(z, led.NoColor)

		// Assert
		for _, actual := range ledLayout[z] {
			assert.Equal(t, byte(0), actual)
		}
		assert.Nil(t, err)
	})

	t.Run("WhenInBoundsAndSingleColor", func(t *testing.T) {
		z := uint8(rand.Intn(8))
		testArgs := [][]any{
			{z, []int{8, 16}, led.Green},
			{z, []int{0, 16}, led.Blue},
			{z, []int{0, 8}, led.Red},
		}

		testCase := func(t *testing.T, z uint8, offsets []int, c led.Color) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetBlock(led.White)

			// Act
			err := ledLayout.ChangeLayer(z, c)

			// Assert
			for _, offset := range offsets {
				for index := offset; index < offset+8; index++ {
					assert.Equal(t, byte(0), ledLayout[z][index])
				}
			}
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndTwoColors", func(t *testing.T) {
		z := uint8(rand.Intn(8))
		testArgs := [][]any{
			{z, 16, led.Cyan},
			{z, 8, led.Yellow},
			{z, 0, led.Violet},
		}

		testCase := func(t *testing.T, z uint8, offset int, c led.Color) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetBlock(led.White)

			// Act
			err := ledLayout.ChangeLayer(z, c)

			// Assert
			for index := offset; index < offset+8; index++ {
				assert.Equal(t, byte(0), ledLayout[z][index])
			}
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndThreeColors", func(t *testing.T) {
		// Arrange
		z := uint8(rand.Intn(8))
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.Red)

		// Act
		err := ledLayout.ChangeLayer(z, led.White)

		// Assert
		for _, actual := range ledLayout[z] {
			assert.Equal(t, byte(255), actual)
		}
		assert.Nil(t, err)
	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		err := ledLayout.ChangeLayer(69, led.Color(uint8(rand.Intn(8))))

		// Assert
		assert.ErrorIs(t, err, common.ErrOutOfBounds)
	})

	t.Run("WhenInBoundsAndColorIsNotPredefined", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		err := ledLayout.ChangeLayer(0, led.Color(uint8(8+rand.Intn(247))))

		// Assert
		assert.Nil(t, err)
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		z := uint8(rand.Intn(8))
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		ledLayout.ChangeLayer(z, led.NoColor)
		ledLayout.ChangeLayer(z, led.Cyan)
		ledLayout.ChangeLayer(z, led.Red)

		// Assert
		for i := 0; i < 15; i++ {
			assert.Equal(t, byte(0), ledLayout[z][i])
		}
		for i := 16; i < 24; i++ {
			assert.Equal(t, byte(255), ledLayout[z][i])
		}
	})
}

func TestLedLayout_ChangeBlock(t *testing.T) {
	t.Run("WhenInBoundsAndNoColor", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		ledLayout.ChangeBlock(led.NoColor)

		// Assert
		for _, layer := range ledLayout {
			for _, actual := range layer {
				assert.Equal(t, byte(0), actual)
			}
		}
	})

	t.Run("WhenInBoundsAndSingleColor", func(t *testing.T) {
		testArgs := [][]any{
			{[]int{8, 16}, led.Green},
			{[]int{0, 16}, led.Blue},
			{[]int{0, 8}, led.Red},
		}

		testCase := func(t *testing.T, offsets []int, c led.Color) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetBlock(led.White)

			// Act
			ledLayout.ChangeBlock(c)

			// Assert
			for _, layer := range ledLayout {
				for _, offset := range offsets {
					for index := offset; index < offset+8; index++ {
						assert.Equal(t, byte(0), layer[index])
					}
				}
			}
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndTwoColors", func(t *testing.T) {
		testArgs := [][]any{
			{16, led.Cyan},
			{8, led.Yellow},
			{0, led.Violet},
		}

		testCase := func(t *testing.T, offset int, c led.Color) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetBlock(led.White)

			// Act
			ledLayout.ChangeBlock(c)

			// Assert
			for _, layer := range ledLayout {
				for index := offset; index < offset+8; index++ {
					assert.Equal(t, byte(0), layer[index])
				}
			}
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndThreeColors", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.Red)

		// Act
		ledLayout.ChangeBlock(led.White)

		// Assert
		for _, layer := range ledLayout {
			for _, actual := range layer {
				assert.Equal(t, byte(255), actual)
			}
		}
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		ledLayout.ChangeBlock(led.NoColor)
		ledLayout.ChangeBlock(led.Cyan)
		ledLayout.ChangeBlock(led.Red)

		// Assert
		for _, layer := range ledLayout {
			for index := 0; index < 16; index++ {
				assert.Equal(t, byte(0), layer[index])
			}
			for index := 16; index < 24; index++ {
				assert.Equal(t, byte(255), layer[index])
			}
		}
	})
}

func TestLedLayout_SetSingle(t *testing.T) {
	t.Run("WhenInBoundsAndSingleColor", func(t *testing.T) {
		x := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(1 << x)
		testArgs := [][]any{
			{x, y, z, y, led.Green, expected},
			{x, y, z, y + 8, led.Blue, expected},
			{x, y, z, y + 16, led.Red, expected},
		}

		testCase := func(t *testing.T, x, y, z, index uint8, c led.Color, expected byte) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.SetSingle(x, y, z, c)

			// Assert
			assert.Equal(t, expected, ledLayout[z][index])
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndTwoColors", func(t *testing.T) {
		x := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(1 << x)
		testArgs := [][]any{
			{x, y, z, []uint8{y, y + 8}, led.Cyan, expected},
			{x, y, z, []uint8{y, y + 16}, led.Yellow, expected},
			{x, y, z, []uint8{y + 8, y + 16}, led.Violet, expected},
		}

		testCase := func(t *testing.T, x, y, z uint8, indexes []uint8, c led.Color, expected byte) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.SetSingle(x, y, z, c)

			// Assert
			for _, index := range indexes {
				assert.Equal(t, expected, ledLayout[z][index])
			}
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndThreeColors", func(t *testing.T) {
		// Arrange
		x := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(1 << x)
		ledLayout := &led.LedLayout{}

		// Act
		err := ledLayout.SetSingle(x, y, z, led.White)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)
	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		testArgs := [][]any{
			{uint8(100), uint8(1), uint8(2)},
			{uint8(3), uint8(200), uint8(4)},
			{uint8(5), uint8(6), uint8(69)},
			{uint8(0), uint8(70), uint8(69)},
			{uint8(101), uint8(5), uint8(69)},
			{uint8(99), uint8(10), uint8(3)},
			{uint8(8), uint8(9), uint8(10)},
		}

		testCase := func(t *testing.T, x, y, z uint8) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.SetSingle(x, y, z, led.Color(uint8(rand.Intn(8))))

			// Assert
			assert.ErrorIs(t, err, common.ErrOutOfBounds)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndColorIsNotPredefined", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		actual := ledLayout.SetSingle(0, 0, 0, led.Color(uint8(8+rand.Intn(247))))

		// Assert
		assert.Nil(t, actual)
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		xFirst := uint8(rand.Intn(8))
		xSecond := uint8(rand.Intn(8))
		xThird := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expectedGreen := byte(1<<xFirst) | byte(1<<xSecond)
		expectedBlue := byte(1<<xFirst) | byte(1<<xSecond) | byte(1<<xThird)
		expectedRed := byte(1 << xFirst)
		ledLayout := &led.LedLayout{}

		// Act
		ledLayout.SetSingle(xFirst, y, z, led.White)
		ledLayout.SetSingle(xSecond, y, z, led.Cyan)
		ledLayout.SetSingle(xThird, y, z, led.Blue)

		// Assert
		assert.Equal(t, expectedGreen, ledLayout[z][y])
		assert.Equal(t, expectedBlue, ledLayout[z][y+8])
		assert.Equal(t, expectedRed, ledLayout[z][y+16])
	})
}

func TestLedLayout_SetRowIndividual(t *testing.T) {
	t.Run("WhenInBoundsAndSingleColor", func(t *testing.T) {
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := uint8(rand.Int())
		testArgs := [][]any{
			{y, z, y, led.Green, expected},
			{y, z, y + 8, led.Blue, expected},
			{y, z, y + 16, led.Red, expected},
		}

		testCase := func(t *testing.T, y, z, index uint8, c led.Color, expected byte) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.SetRowIndividual(y, z, c, expected)

			// Assert
			assert.Equal(t, expected, ledLayout[z][index])
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndTwoColors", func(t *testing.T) {
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := uint8(rand.Int())
		testArgs := [][]any{
			{y, z, []uint8{y, y + 8}, led.Cyan, expected},
			{y, z, []uint8{y, y + 16}, led.Yellow, expected},
			{y, z, []uint8{y + 8, y + 16}, led.Violet, expected},
		}

		testCase := func(t *testing.T, y, z uint8, indexes []uint8, c led.Color, expected byte) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.SetRowIndividual(y, z, c, expected)

			// Assert
			for _, index := range indexes {
				assert.Equal(t, expected, ledLayout[z][index])
			}
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndThreeColors", func(t *testing.T) {
		// Arrange
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := uint8(rand.Int())
		ledLayout := &led.LedLayout{}

		// Act
		err := ledLayout.SetRowIndividual(y, z, led.White, expected)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)
	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		testArgs := [][]any{
			{uint8(100), uint8(1)},
			{uint8(2), uint8(200)},
			{uint8(69), uint8(70)},
		}

		testCase := func(t *testing.T, y, z uint8) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.SetRowIndividual(y, z, led.Color(uint8(rand.Intn(8))), uint8(rand.Int()))

			// Assert
			assert.ErrorIs(t, err, common.ErrOutOfBounds)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndColorIsNotPredefined", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		err := ledLayout.SetRowIndividual(0, 0, led.Color(uint8(8+rand.Intn(247))), uint8(rand.Int()))

		// Assert
		assert.Nil(t, err)
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		valueFirst := uint8(rand.Int())
		valueSecond := uint8(rand.Int())
		valueThird := uint8(rand.Int())
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expectedGreen := valueFirst | valueSecond
		expectedBlue := valueFirst | valueSecond | valueThird
		expectedRed := valueFirst
		ledLayout := &led.LedLayout{}

		// Act
		ledLayout.SetRowIndividual(y, z, led.White, valueFirst)
		ledLayout.SetRowIndividual(y, z, led.Cyan, valueSecond)
		ledLayout.SetRowIndividual(y, z, led.Blue, valueThird)

		// Assert
		assert.Equal(t, expectedGreen, ledLayout[z][y])
		assert.Equal(t, expectedBlue, ledLayout[z][y+8])
		assert.Equal(t, expectedRed, ledLayout[z][y+16])
	})
}

func TestLedLayout_SetRow(t *testing.T) {
	t.Run("WhenInBoundsAndSingleColor", func(t *testing.T) {
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		testArgs := [][]any{
			{y, z, y, led.Green},
			{y, z, y + 8, led.Blue},
			{y, z, y + 16, led.Red},
		}

		testCase := func(t *testing.T, y, z, index uint8, c led.Color) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.SetRow(y, z, c)

			// Assert
			assert.Equal(t, byte(255), ledLayout[z][index])
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndTwoColors", func(t *testing.T) {
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		testArgs := [][]any{
			{y, z, []uint8{y, y + 8}, led.Cyan},
			{y, z, []uint8{y, y + 16}, led.Yellow},
			{y, z, []uint8{y + 8, y + 16}, led.Violet},
		}

		testCase := func(t *testing.T, y, z uint8, indexes []uint8, c led.Color) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.SetRow(y, z, c)

			// Assert
			for _, index := range indexes {
				assert.Equal(t, byte(255), ledLayout[z][index])
			}
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndThreeColors", func(t *testing.T) {
		// Arrange
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(255)
		ledLayout := &led.LedLayout{}

		// Act
		err := ledLayout.SetRow(y, z, led.White)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)
	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		testArgs := [][]any{
			{uint8(100), uint8(1)},
			{uint8(2), uint8(200)},
			{uint8(68), uint8(69)},
		}

		testCase := func(t *testing.T, y, z uint8) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.SetRow(y, z, led.Color(uint8(rand.Intn(8))))

			// Assert
			assert.ErrorIs(t, err, common.ErrOutOfBounds)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndColorIsNotPredefined", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		actual := ledLayout.SetRow(0, 0, led.Color(uint8(8+rand.Intn(247))))

		// Assert
		assert.Nil(t, actual)
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := byte(255)
		ledLayout := &led.LedLayout{}

		// Act
		ledLayout.SetRow(y, z, led.Cyan)
		ledLayout.SetRow(y, z, led.Red)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
	})
}

func TestLedLayout_SetLayer(t *testing.T) {
	t.Run("WhenInBoundsAndSingleColor", func(t *testing.T) {
		z := uint8(rand.Intn(8))
		testArgs := [][]any{
			{z, 0, led.Green},
			{z, 8, led.Blue},
			{z, 16, led.Red},
		}

		testCase := func(t *testing.T, z uint8, offset int, c led.Color) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.SetLayer(z, c)

			// Assert
			for index := offset; index < offset+8; index++ {
				assert.Equal(t, byte(255), ledLayout[z][index])
			}
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndTwoColors", func(t *testing.T) {
		z := uint8(rand.Intn(8))
		testArgs := [][]any{
			{z, []int{0, 8}, led.Cyan},
			{z, []int{0, 16}, led.Yellow},
			{z, []int{8, 16}, led.Violet},
		}

		testCase := func(t *testing.T, z uint8, indexes []int, c led.Color) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			err := ledLayout.SetLayer(z, c)

			// Assert
			for _, offset := range indexes {
				for index := offset; index < offset+8; index++ {
					assert.Equal(t, byte(255), ledLayout[z][index])
				}
			}
			assert.Nil(t, err)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenInBoundsAndThreeColors", func(t *testing.T) {
		// Arrange
		z := uint8(rand.Intn(8))
		ledLayout := &led.LedLayout{}

		// Act
		err := ledLayout.SetLayer(z, led.White)

		// Assert
		for _, actual := range ledLayout[z] {
			assert.Equal(t, byte(255), actual)
		}
		assert.Nil(t, err)
	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		err := ledLayout.SetLayer(69, led.Color(uint8(rand.Intn(8))))

		// Assert
		assert.ErrorIs(t, err, common.ErrOutOfBounds)
	})

	t.Run("WhenInBoundsAndColorIsNotPredefined", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		actual := ledLayout.SetLayer(0, led.Color(uint8(8+rand.Intn(247))))

		// Assert
		assert.Nil(t, actual)
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		z := uint8(rand.Intn(8))
		ledLayout := &led.LedLayout{}

		// Act
		ledLayout.SetLayer(z, led.Yellow)
		ledLayout.SetLayer(z, led.Blue)

		// Assert
		for _, actual := range ledLayout[z] {
			assert.Equal(t, byte(255), actual)
		}
	})
}

func TestLedLayout_SetBlock(t *testing.T) {
	t.Run("WhenCaledLayoutedWithSingleColor", func(t *testing.T) {
		testArgs := [][]any{
			{led.Green, 0},
			{led.Blue, 8},
			{led.Red, 16},
		}

		testCase := func(t *testing.T, c led.Color, offset int) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			ledLayout.SetBlock(c)

			// Assert
			for _, layer := range ledLayout {
				for index := offset; index < offset+8; index++ {
					assert.Equal(t, byte(255), layer[index])
				}
			}
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenCaledLayoutedWithTwoColors", func(t *testing.T) {
		testArgs := [][]any{
			{led.Cyan, []int{0, 8}},
			{led.Yellow, []int{0, 16}},
			{led.Violet, []int{8, 16}},
		}

		testCase := func(t *testing.T, c led.Color, offsets []int) {
			// Arrange
			ledLayout := &led.LedLayout{}

			// Act
			ledLayout.SetBlock(c)

			// Assert
			for _, layer := range ledLayout {
				for _, offset := range offsets {
					for index := offset; index < offset+8; index++ {
						assert.Equal(t, byte(255), layer[index])
					}
				}
			}
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenCaledLayoutedWithThreeColors", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		ledLayout.SetBlock(led.White)

		// Assert
		for _, layer := range ledLayout {
			for _, state := range layer {
				assert.Equal(t, byte(255), state)
			}
		}
	})

	t.Run("WhenCaledLayoutedMultipleTimes", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}

		// Act
		ledLayout.SetBlock(led.Red)
		ledLayout.SetBlock(led.Violet)
		ledLayout.SetBlock(led.Cyan)
		ledLayout.SetBlock(led.Green)

		// Assert
		for _, layer := range ledLayout {
			for _, state := range layer {
				assert.Equal(t, byte(255), state)
			}
		}
	})
}

func TestLedLayout_ResetSingle(t *testing.T) {
	t.Run("WhenInBounds", func(t *testing.T) {
		// Arrange
		x := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := ^byte(1 << x)
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		err := ledLayout.ResetSingle(x, y, z)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)
	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		testArgs := [][]any{
			{uint8(100), uint8(1), uint8(2)},
			{uint8(3), uint8(200), uint8(4)},
			{uint8(5), uint8(6), uint8(69)},
			{uint8(0), uint8(70), uint8(69)},
			{uint8(101), uint8(5), uint8(69)},
			{uint8(99), uint8(10), uint8(3)},
			{uint8(8), uint8(9), uint8(10)},
		}

		testCase := func(t *testing.T, x, y, z uint8) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetBlock(led.White)

			// Act
			err := ledLayout.ResetSingle(x, y, z)

			// Assert
			assert.ErrorIs(t, err, common.ErrOutOfBounds)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		xFirst := uint8(rand.Intn(8))
		xSecond := uint8(rand.Intn(8))
		xThird := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := ^byte(1<<xFirst) & ^byte(1<<xSecond) & ^byte(1<<xThird)
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		ledLayout.ResetSingle(xFirst, y, z)
		ledLayout.ResetSingle(xSecond, y, z)
		ledLayout.ResetSingle(xThird, y, z)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
	})
}

func TestLedLayout_ResetRowIndividual(t *testing.T) {
	t.Run("WhenInBounds", func(t *testing.T) {
		// Arrange
		x := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := ^byte(1 << x)
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		err := ledLayout.ResetSingle(x, y, z)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)

	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		testArgs := [][]any{
			{uint8(100), uint8(1), uint8(2)},
			{uint8(3), uint8(200), uint8(4)},
			{uint8(5), uint8(6), uint8(69)},
			{uint8(0), uint8(70), uint8(69)},
			{uint8(101), uint8(5), uint8(69)},
			{uint8(99), uint8(10), uint8(3)},
			{uint8(8), uint8(9), uint8(10)},
		}

		testCase := func(t *testing.T, x, y, z uint8) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetBlock(led.White)

			// Act
			err := ledLayout.ResetSingle(x, y, z)

			// Assert
			assert.ErrorIs(t, err, common.ErrOutOfBounds)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		xFirst := uint8(rand.Intn(8))
		xSecond := uint8(rand.Intn(8))
		xThird := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := ^byte(1<<xFirst) & ^byte(1<<xSecond) & ^byte(1<<xThird)
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		ledLayout.ResetSingle(xFirst, y, z)
		ledLayout.ResetSingle(xSecond, y, z)
		ledLayout.ResetSingle(xThird, y, z)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
	})
}

func TestLedLayout_ResetRow(t *testing.T) {
	t.Run("WhenInBounds", func(t *testing.T) {
		// Arrange
		x := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := ^byte(1 << x)
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		err := ledLayout.ResetSingle(x, y, z)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
		assert.Nil(t, err)

	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		testArgs := [][]any{
			{uint8(100), uint8(1), uint8(2)},
			{uint8(3), uint8(200), uint8(4)},
			{uint8(5), uint8(6), uint8(69)},
			{uint8(0), uint8(70), uint8(69)},
			{uint8(101), uint8(5), uint8(69)},
			{uint8(99), uint8(10), uint8(3)},
			{uint8(8), uint8(9), uint8(10)},
		}

		testCase := func(t *testing.T, x, y, z uint8) {
			// Arrange
			ledLayout := &led.LedLayout{}
			ledLayout.SetBlock(led.White)

			// Act
			err := ledLayout.ResetSingle(x, y, z)

			// Assert
			assert.ErrorIs(t, err, common.ErrOutOfBounds)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenCaledLayoutedMulitpleTimes", func(t *testing.T) {
		// Arrange
		xFirst := uint8(rand.Intn(8))
		xSecond := uint8(rand.Intn(8))
		xThird := uint8(rand.Intn(8))
		y := uint8(rand.Intn(8))
		z := uint8(rand.Intn(8))
		expected := ^byte(1<<xFirst) & ^byte(1<<xSecond) & ^byte(1<<xThird)
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		ledLayout.ResetSingle(xFirst, y, z)
		ledLayout.ResetSingle(xSecond, y, z)
		ledLayout.ResetSingle(xThird, y, z)

		// Assert
		assert.Equal(t, expected, ledLayout[z][y])
		assert.Equal(t, expected, ledLayout[z][y+8])
		assert.Equal(t, expected, ledLayout[z][y+16])
	})
}

func TestLedLayout_ResetLayer(t *testing.T) {
	t.Run("WhenInBounds", func(t *testing.T) {
		// Arrange
		layer := uint8(rand.Intn(8))
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		err := ledLayout.ResetLayer(layer)

		// Assert
		for _, state := range ledLayout[layer] {
			assert.Equal(t, byte(0), state)
		}
		assert.Nil(t, err)

	})

	t.Run("WhenOutOfBounds", func(t *testing.T) {
		// Arrange
		ledLayout := &led.LedLayout{}
		ledLayout.SetBlock(led.White)

		// Act
		err := ledLayout.ResetLayer(uint8(8 + rand.Intn(247)))

		// Assert
		assert.ErrorIs(t, err, common.ErrOutOfBounds)
	})
}

func TestLedLayout_ResetBlock(t *testing.T) {
	// Arrange
	ledLayout := &led.LedLayout{}
	ledLayout.SetBlock(led.White)

	// Act
	ledLayout.ResetBlock()

	// Assert
	for _, layer := range ledLayout {
		for _, state := range layer {
			assert.Equal(t, byte(0), state)
		}
	}
}
