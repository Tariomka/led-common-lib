package uart_test

import (
	"strings"
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/communication/uart"
	"github.com/stretchr/testify/assert"
)

type received struct {
	lines  []string
	frames [][]byte
}

func feed(reader *uart.Reader, data []byte, now uint32) received {
	var result received
	for _, b := range data {
		switch item, value := reader.Feed(b, now); item {
		case uart.Line:
			result.lines = append(result.lines, string(value))
		case uart.Frame:
			result.frames = append(result.frames, append([]byte(nil), value...))
		}
	}
	return result
}

var readyFrame = []byte{0xA5, 0x05, 0x06, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0xE3, 0x55}

func TestAppendFrame(t *testing.T) {
	// Arrange
	// Act
	actual := uart.AppendFrame(nil, []byte{16, 0, 0, 0, 0, 0})

	// Assert
	assert.Equal(t, readyFrame, actual)
}

func TestReaderFeed(t *testing.T) {
	t.Run("WhenTextLinesArrive", func(t *testing.T) {
		// Arrange
		reader := &uart.Reader{}

		// Act
		actual := feed(reader, []byte("Hello\r\nWorld\n\r\n"), 0)

		// Assert
		assert.Equal(t, []string{"Hello", "World", ""}, actual.lines)
		assert.Empty(t, actual.frames)
	})

	t.Run("WhenFrameInterleavesWithText", func(t *testing.T) {
		// Arrange
		reader := &uart.Reader{}
		data := append([]byte("Sh"), readyFrame...)
		data = append(data, "ow\r\n"...)

		// Act
		actual := feed(reader, data, 0)

		// Assert
		assert.Equal(t, []string{"Show"}, actual.lines)
		assert.Equal(t, [][]byte{{16, 0, 0, 0, 0, 0}}, actual.frames)
	})

	t.Run("WhenCheckIsWrong", func(t *testing.T) {
		// Arrange
		reader := &uart.Reader{}
		data := append([]byte(nil), readyFrame...)
		data[9] ^= 1
		data = append(data, readyFrame...)

		// Act
		actual := feed(reader, data, 0)

		// Assert
		assert.Len(t, actual.frames, 1)
	})

	t.Run("WhenFrameHasAGap", func(t *testing.T) {
		// Arrange
		reader := &uart.Reader{}

		// Act
		first := feed(reader, readyFrame[:4], 0)
		second := feed(reader, readyFrame[4:], 1001)
		third := feed(reader, readyFrame, 1002)

		// Assert
		assert.Empty(t, first.frames)
		assert.Empty(t, second.frames)
		assert.Len(t, third.frames, 1)
	})

	t.Run("WhenLineIsTooLong", func(t *testing.T) {
		// Arrange
		reader := &uart.Reader{}

		// Act
		actual := feed(reader, []byte(strings.Repeat("x", 300)+"\r\n"), 0)

		// Assert
		assert.Equal(t, []string{strings.Repeat("x", uart.MaxLine)}, actual.lines)
	})
}
