package uart_test

import (
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/common"
	"github.com/Tariomka/led-common-lib/pkg/communication/uart"
	"github.com/Tariomka/led-common-lib/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCube answers upload frames like the cube's firmware. Its output is queued and delivered by
// pump, like a host loop reading the port.
type fakeCube struct {
	reader    uart.Reader
	flash     []byte
	lines     []uint16
	ends      int
	ready     byte
	dropReply map[uint16]int // record line -> replies to drop
	silentEnd bool
	failEnd   bool
	output    []byte
}

func newFakeCube() *fakeCube {
	return &fakeCube{flash: make([]byte, uart.ShowCapacity), dropReply: map[uint16]int{}}
}

func (this *fakeCube) Write(data []byte) (int, error) {
	for _, b := range data {
		if item, payload := this.reader.Feed(b, 0); item == uart.Frame {
			this.frame(payload)
		}
	}
	return len(data), nil
}

func (this *fakeCube) frame(payload []byte) {
	switch payload[0] {
	case 17:
		this.output = append(this.output, "Show upload started\r\n"...)
		this.output = uart.AppendFrame(this.output, []byte{16, this.ready, 0, 0, 0, 0})
	case 32:
		line := uint16(payload[2])<<8 | uint16(payload[3])
		address := int(payload[4])<<24 | int(payload[5])<<16 | int(payload[6])<<8 | int(payload[7])
		copy(this.flash[address-uart.ShowAddress:], payload[8:])
		this.lines = append(this.lines, line)
		if this.dropReply[line] > 0 {
			this.dropReply[line]--
			return
		}
		this.output = uart.AppendFrame(this.output, []byte{33, payload[1], payload[2], payload[3]})
	case 37:
		this.ends++
		if this.ends == 1 && !this.silentEnd {
			if this.failEnd {
				this.output = append(this.output, "Show upload failed\r\n"...)
			} else {
				this.output = append(this.output, "Show upload OK\r\n"...)
			}
		}
	}
}

// pump delivers the cube's output to the upload client until there is nothing left.
func pump(cube *fakeCube, upload *uart.Upload, reader *uart.Reader, now uint32) {
	for len(cube.output) > 0 {
		output := cube.output
		cube.output = nil
		for _, b := range output {
			switch item, data := reader.Feed(b, now); item {
			case uart.Line:
				upload.Line(string(data))
			case uart.Frame:
				upload.Frame(data, now)
			}
		}
	}
}

// send uploads data in chunks, waiting for each chunk to be stored like the Pico does.
func send(t *testing.T, cube *fakeCube, upload *uart.Upload, data []byte, chunk int) {
	reader := &uart.Reader{}
	require.NoError(t, upload.Begin(len(data), 0))
	pump(cube, upload, reader, 0)
	for offset := 0; offset < len(data); offset += chunk {
		end := min(offset+chunk, len(data))
		require.Equal(t, offset, upload.Queued())
		require.NoError(t, upload.Write(data[offset:end], 0))
		for now := uint32(0); upload.Acked() < end && upload.State() == uart.UploadSending; now += uart.Timeout {
			upload.Tick(now)
			pump(cube, upload, reader, now)
		}
	}
	if upload.State() == uart.UploadSending {
		require.NoError(t, upload.Finish(0))
		pump(cube, upload, reader, 0)
	}
}

func TestUpload(t *testing.T) {
	t.Run("WhenShowDataIsSent", func(t *testing.T) {
		// Arrange
		data := []byte(helpers.OriginalShow(t))
		cube := newFakeCube()
		upload := uart.NewUpload(cube)

		// Act
		send(t, cube, upload, data, 2048)

		// Assert
		assert.Equal(t, uart.UploadDone, upload.State())
		assert.NoError(t, upload.Err())
		assert.Equal(t, len(data), upload.Acked())
		assert.Equal(t, len(data), upload.Total())
		assert.Equal(t, data, cube.flash[:len(data)])
		assert.Equal(t, 2, cube.ends)
		for i, line := range cube.lines {
			require.Equal(t, uint16(i), line)
		}
	})

	t.Run("WhenReplyIsLost", func(t *testing.T) {
		// Arrange
		data := []byte(helpers.OriginalShow(t))[:1000]
		cube := newFakeCube()
		cube.dropReply[3] = 1
		upload := uart.NewUpload(cube)

		// Act
		send(t, cube, upload, data, 500)

		// Assert
		assert.Equal(t, uart.UploadDone, upload.State())
		assert.Equal(t, data, cube.flash[:len(data)])
		assert.Equal(t, []uint16{0, 1, 2, 3, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, cube.lines)
	})

	t.Run("WhenReplyIsLostTwice", func(t *testing.T) {
		// Arrange
		cube := newFakeCube()
		cube.dropReply[1] = 2
		upload := uart.NewUpload(cube)

		// Act
		send(t, cube, upload, make([]byte, 1000), 1000)

		// Assert
		assert.Equal(t, uart.UploadFailed, upload.State())
		assert.ErrorIs(t, upload.Err(), common.ErrUploadTimeout)
		assert.Equal(t, uart.RecordSize, upload.Acked())
		assert.Zero(t, cube.ends)
	})

	t.Run("WhenCubeRefuses", func(t *testing.T) {
		// Arrange
		cube := newFakeCube()
		cube.ready = 1
		upload := uart.NewUpload(cube)

		// Act
		require.NoError(t, upload.Begin(10, 0))
		pump(cube, upload, &uart.Reader{}, 0)

		// Assert
		assert.Equal(t, uart.UploadFailed, upload.State())
		assert.ErrorIs(t, upload.Err(), common.ErrUploadRefused)
	})

	t.Run("WhenCubeReportsFailure", func(t *testing.T) {
		// Arrange
		cube := newFakeCube()
		cube.failEnd = true
		upload := uart.NewUpload(cube)

		// Act
		send(t, cube, upload, make([]byte, 300), 300)

		// Assert
		assert.Equal(t, uart.UploadFailed, upload.State())
		assert.ErrorIs(t, upload.Err(), common.ErrUploadFailed)
	})

	t.Run("WhenResultLineIsMissing", func(t *testing.T) {
		// Arrange
		cube := newFakeCube()
		cube.silentEnd = true
		upload := uart.NewUpload(cube)
		send(t, cube, upload, make([]byte, 10), 10)

		// Act
		upload.Tick(uart.Timeout)

		// Assert
		assert.Equal(t, uart.UploadFailed, upload.State())
		assert.ErrorIs(t, upload.Err(), common.ErrUploadTimeout)
	})

	t.Run("WhenDataArrivesBeforeTheCubeIsReady", func(t *testing.T) {
		// Arrange
		cube := newFakeCube()
		upload := uart.NewUpload(cube)
		reader := &uart.Reader{}
		data := []byte("0123456789")

		// Act
		require.NoError(t, upload.Begin(0, 0))
		require.NoError(t, upload.Write(data, 0))
		require.NoError(t, upload.Finish(0))
		pump(cube, upload, reader, 0)

		// Assert
		assert.Equal(t, uart.UploadDone, upload.State())
		assert.Equal(t, data, cube.flash[:len(data)])
	})
}

func TestUploadErrors(t *testing.T) {
	t.Run("WhenSizeIsTooLarge", func(t *testing.T) {
		// Arrange
		upload := uart.NewUpload(newFakeCube())

		// Act
		err := upload.Begin(uart.ShowCapacity+1, 0)

		// Assert
		assert.ErrorIs(t, err, common.ErrUploadSize)
	})

	t.Run("WhenUploadIsRunning", func(t *testing.T) {
		// Arrange
		upload := uart.NewUpload(newFakeCube())
		require.NoError(t, upload.Begin(10, 0))

		// Act
		err := upload.Begin(10, 0)

		// Assert
		assert.ErrorIs(t, err, common.ErrUploadBusy)
	})

	t.Run("WhenWritingAfterFinish", func(t *testing.T) {
		// Arrange
		upload := uart.NewUpload(newFakeCube())
		require.NoError(t, upload.Begin(10, 0))
		require.NoError(t, upload.Finish(0))

		// Act
		err := upload.Write([]byte{1}, 0)

		// Assert
		assert.ErrorIs(t, err, common.ErrUploadState)
	})

	t.Run("WhenWritingWithoutBegin", func(t *testing.T) {
		// Arrange
		upload := uart.NewUpload(newFakeCube())

		// Act
		err := upload.Write([]byte{1}, 0)

		// Assert
		assert.ErrorIs(t, err, common.ErrUploadState)
	})
}
