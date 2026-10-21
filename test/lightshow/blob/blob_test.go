package blob_test

import (
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/common"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/blob"
	"github.com/Tariomka/led-common-lib/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseOriginal(t *testing.T) *blob.Blob {
	t.Helper()
	b, err := blob.Parse(helpers.OriginalShow(t))
	require.NoError(t, err)
	return b
}

func TestParse(t *testing.T) {
	t.Run("WhenDataIsTheFactoryShow", func(t *testing.T) {
		// Arrange
		// Act
		b := parseOriginal(t)

		// Assert
		assert.Equal(t, 32100, b.Size())
		for key, want := range []int{2, 3, 4, 6, 8, 7, 9} {
			assert.Equal(t, want, b.KeyFunction(key), "KEY%d", key+1)
		}
		assert.Equal(t, 1, b.DisplayCount())
		assert.Equal(t, blob.Display{CubeType: 1}, b.Display(0))

		palette, ok := b.Palette("red-series")
		assert.True(t, ok)
		assert.Equal(t, "\x00\x30\x31\x32\x33\x3B\x3F\x00\x00\x00\x00\x00\x00\x00\x00\x00", palette)

		graphs := 0
		b.ForEachGraph(func(blob.Graph) { graphs++ })
		assert.Equal(t, 79, graphs)

		index, ok := b.Index("TwoSide")
		assert.True(t, ok)
		assert.Len(t, index, 60)
		assert.Equal(t, 7, blob.IndexValue(index, 0, 1))

		for group, want := range []int{41, 10, 4, 1} {
			assert.Len(t, b.Group(group), want, "group %d", group+1)
		}
		assert.Equal(t, blob.FrameHeader{FrameCount: 102, Loop: 1, Delay: 150}, b.Group(3)[0].Header())
	})

	t.Run("WhenMagicIsWrong", func(t *testing.T) {
		// Arrange
		data := []byte(helpers.OriginalShow(t))
		data[0] = 0

		// Act
		_, err := blob.Parse(string(data))

		// Assert
		assert.ErrorIs(t, err, common.ErrBlobMagic)
	})

	t.Run("WhenSectionIsOutsideData", func(t *testing.T) {
		// Arrange
		data := helpers.OriginalShow(t)[:1000]

		// Act
		_, err := blob.Parse(data)

		// Assert
		assert.ErrorIs(t, err, common.ErrBlobLayout)
	})
}

func TestGraphDecode(t *testing.T) {
	// Arrange
	b := parseOriginal(t)

	// Act
	// Assert
	b.ForEachGraph(func(graph blob.Graph) {
		voxels := make([]byte, graph.Voxels())
		assert.NoError(t, graph.Decode(voxels), graph.Name)
	})
}

func TestResolveName(t *testing.T) {
	testArgs := []struct {
		name         string
		frame, start int
		expected     string
	}{
		{"plain", 3, 0, "plain"},
		{"A$F", 7, 0, "A7"},
		{"Fly_$C%4+1", 5, 0, "Fly_2"},
		{"A$C-2*3", 10, 4, "A12"},
		{"A$X", 1, 0, ""},
		{"A$F/0", 1, 0, ""},
		{"A$F+", 1, 0, ""},
	}

	for _, args := range testArgs {
		t.Run(args.name, func(t *testing.T) {
			// Arrange
			// Act
			actual := blob.ResolveName(args.name, args.frame, args.start)

			// Assert
			assert.Equal(t, args.expected, actual)
		})
	}
}
