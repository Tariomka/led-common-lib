package tcp_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/common"
	"github.com/Tariomka/led-common-lib/pkg/communication/tcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	t.Run("WhenMessageHasPayload", func(t *testing.T) {
		// Arrange
		message := tcp.NewCommand("Status")

		// Act
		actual, err := message.Marshal()

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, []byte{tcp.Version, 0x02, 0x00, 0x06, 'S', 't', 'a', 't', 'u', 's'}, actual)
	})

	t.Run("WhenMessageHasNoPayload", func(t *testing.T) {
		// Arrange
		// Act
		actual, err := tcp.NewPing().Marshal()

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, []byte{tcp.Version, 0x20, 0x00, 0x00}, actual)
	})

	t.Run("WhenPayloadIsTooLarge", func(t *testing.T) {
		// Arrange
		message := tcp.NewCubeLine(string(make([]byte, tcp.MaxPayload+1)))

		// Act
		_, err := message.Marshal()

		// Assert
		assert.ErrorIs(t, err, common.ErrPayloadTooLarge)
	})
}

func TestRead(t *testing.T) {
	t.Run("WhenMessagesFollowEachOther", func(t *testing.T) {
		// Arrange
		messages := []tcp.Message{
			tcp.NewHello("desktop"),
			tcp.NewCommand("Key 2"),
			tcp.NewCubeLine("Next Cartoon"),
			tcp.NewPicoLine("Wi-Fi connected"),
			tcp.NewUploadBegin(32100),
			tcp.NewUploadChunk(2048, []byte{1, 2, 3}),
			tcp.NewUploadEnd(),
			tcp.NewUploadProgress(4096, 32100),
			tcp.NewUploadResult(true, "Show upload OK"),
			tcp.NewPing(),
			tcp.NewPong(),
		}
		stream := &bytes.Buffer{}
		for _, message := range messages {
			require.NoError(t, tcp.Write(stream, message))
		}
		buffer := make([]byte, 4+tcp.MaxChunk)

		// Act
		// Assert
		for _, expected := range messages {
			actual, err := tcp.Read(stream, buffer)
			require.NoError(t, err)
			assert.Equal(t, expected.Version, actual.Version)
			assert.Equal(t, expected.Type, actual.Type)
			assert.Equal(t, len(expected.Payload), len(actual.Payload))
			assert.Equal(t, string(expected.Payload), string(actual.Payload))
		}
		_, err := tcp.Read(stream, buffer)
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("WhenVersionIsDifferent", func(t *testing.T) {
		// Arrange
		stream := bytes.NewBuffer([]byte{tcp.Version + 1, 0x7F, 0x00, 0x02, 0xAB, 0xCD, tcp.Version, 0x20, 0x00, 0x00})
		buffer := make([]byte, 16)

		// Act
		future, futureErr := tcp.Read(stream, buffer)
		ping, pingErr := tcp.Read(stream, buffer)

		// Assert
		assert.ErrorIs(t, futureErr, common.ErrUnsupportedVersion)
		assert.Equal(t, tcp.Message{Version: tcp.Version + 1, Type: 0x7F, Payload: []byte{0xAB, 0xCD}}, future)
		assert.NoError(t, pingErr)
		assert.Equal(t, tcp.TypePing, ping.Type)
	})

	t.Run("WhenPayloadDoesNotFitTheBuffer", func(t *testing.T) {
		// Arrange
		stream := &bytes.Buffer{}
		require.NoError(t, tcp.Write(stream, tcp.NewCubeLine("too long for the buffer")))
		require.NoError(t, tcp.Write(stream, tcp.NewPong()))
		buffer := make([]byte, 8)

		// Act
		skipped, skippedErr := tcp.Read(stream, buffer)
		pong, pongErr := tcp.Read(stream, buffer)

		// Assert
		assert.ErrorIs(t, skippedErr, common.ErrPayloadTooLarge)
		assert.Equal(t, tcp.TypeCubeLine, skipped.Type)
		assert.NoError(t, pongErr)
		assert.Equal(t, tcp.TypePong, pong.Type)
	})

	t.Run("WhenStreamEndsInsideMessage", func(t *testing.T) {
		// Arrange
		stream := bytes.NewBuffer([]byte{tcp.Version, 0x03, 0x00, 0x05, 'a'})

		// Act
		_, err := tcp.Read(stream, make([]byte, 16))

		// Assert
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})
}

func TestPayload(t *testing.T) {
	t.Run("WhenPayloadsAreValid", func(t *testing.T) {
		// Arrange
		// Act
		size, sizeErr := tcp.NewUploadBegin(32100).Size()
		offset, data, chunkErr := tcp.NewUploadChunk(4096, []byte{9, 8, 7}).Chunk()
		acked, total, progressErr := tcp.NewUploadProgress(128, 32100).Progress()
		ok, text, resultErr := tcp.NewUploadResult(false, "Show upload failed").Result()

		// Assert
		assert.NoError(t, sizeErr)
		assert.Equal(t, uint32(32100), size)
		assert.NoError(t, chunkErr)
		assert.Equal(t, uint32(4096), offset)
		assert.Equal(t, []byte{9, 8, 7}, data)
		assert.NoError(t, progressErr)
		assert.Equal(t, uint32(128), acked)
		assert.Equal(t, uint32(32100), total)
		assert.NoError(t, resultErr)
		assert.False(t, ok)
		assert.Equal(t, "Show upload failed", text)
		assert.Equal(t, "Status", tcp.NewCommand("Status").Text())
	})

	t.Run("WhenPayloadsAreTooShort", func(t *testing.T) {
		// Arrange
		empty := tcp.Message{Version: tcp.Version}

		// Act
		_, sizeErr := empty.Size()
		_, _, chunkErr := empty.Chunk()
		_, _, progressErr := empty.Progress()
		_, _, resultErr := empty.Result()

		// Assert
		assert.ErrorIs(t, sizeErr, common.ErrNotEnoughData)
		assert.ErrorIs(t, chunkErr, common.ErrNotEnoughData)
		assert.ErrorIs(t, progressErr, common.ErrNotEnoughData)
		assert.ErrorIs(t, resultErr, common.ErrNotEnoughData)
	})
}
