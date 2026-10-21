package lightshow_test

import (
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/lightshow"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/blob"
	"github.com/Tariomka/led-common-lib/pkg/lightshow/render"
	"github.com/Tariomka/led-common-lib/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCanvas() *render.Canvas {
	return &render.Canvas{Displays: []render.Display{render.NewDisplay(0, 0, 0, 8, 8, 8)}}
}

func originalShow(t *testing.T, group int) *lightshow.BlobShow {
	t.Helper()
	b, err := blob.Parse(helpers.OriginalShow(t))
	require.NoError(t, err)
	return lightshow.NewBlobShow(b, group)
}

func compile(t *testing.T, show *blob.Show) *blob.Blob {
	t.Helper()
	data, err := show.Marshal()
	require.NoError(t, err)
	b, err := blob.Parse(string(data))
	require.NoError(t, err)
	return b
}

func lit(voxels []byte) bool {
	for _, voxel := range voxels {
		if voxel != 0 {
			return true
		}
	}
	return false
}

func TestPlayerStep(t *testing.T) {
	t.Run("WhenPlayingTheTextCartoon", func(t *testing.T) {
		// Arrange
		canvas := newCanvas()
		player := lightshow.New(originalShow(t, 3), canvas)
		litFrames := 0

		// Act
		for frame := 0; frame < 102; frame++ {
			rendered, wait := player.Step(0)

			// Assert
			require.True(t, rendered)
			require.Equal(t, 150, wait)
			if lit(canvas.Displays[0].Voxels) {
				litFrames++
			}
		}
		assert.Greater(t, litFrames, 50)
		assert.Equal(t, 0, player.Current())
	})

	t.Run("WhenCartoonEnds", func(t *testing.T) {
		// Arrange
		show := originalShow(t, 0)
		player := lightshow.New(show, newCanvas())
		header := show.Cartoon(0).Header()

		// Act
		for frame := 0; frame < header.FrameCount*header.Loop; frame++ {
			player.Step(0)
		}
		player.Step(0)

		// Assert
		assert.Equal(t, 1, player.Current())
	})

	t.Run("WhenSpeedIsHighest", func(t *testing.T) {
		// Arrange
		player := lightshow.New(originalShow(t, 3), newCanvas())
		player.SetSpeed(7)

		// Act
		_, wait := player.Step(0)

		// Assert
		assert.Equal(t, 50, wait)
	})

	t.Run("WhenPaused", func(t *testing.T) {
		// Arrange
		player := lightshow.New(originalShow(t, 3), newCanvas())
		player.Paused = true

		// Act
		rendered, _ := player.Step(0)

		// Assert
		assert.False(t, rendered)
	})
}

func TestPlayerAction(t *testing.T) {
	testArgs := []struct {
		name    string
		actions []int
		log     []string
		current int
	}{
		{"Next", []int{lightshow.ActionNext}, []string{"Next Cartoon"}, 1},
		{"PreviousWraps", []int{lightshow.ActionPrevious}, []string{"Previous Cartoon"}, 40},
		{"Replay", []int{lightshow.ActionReplay}, []string{"Replay Cartoon"}, 0},
		{"Jump", []int{lightshow.ActionJumpCartoon + 3}, []string{"Change Cartoon 3"}, 3},
		{"SpeedUp", []int{lightshow.ActionSpeedUp, lightshow.ActionSpeedUp, lightshow.ActionSpeedUp, lightshow.ActionSpeedUp},
			[]string{"Add speed to 1.50", "Add speed to 2.00", "Add speed to 3.00"}, 0},
		{"SpeedDownAndReset", []int{lightshow.ActionSpeedDown, lightshow.ActionSpeedReset, lightshow.ActionSpeedReset},
			[]string{"Sub speed to 0.70", "Speed reset to 1.00"}, 0},
		{"Toggles", []int{lightshow.ActionPlayModel, lightshow.ActionPause, lightshow.ActionPause, lightshow.ActionDisplayOff},
			[]string{"Play Model=0", "DeviceSuppend:1", "DeviceSuppend:0", "Display off=1"}, 0},
	}

	for _, args := range testArgs {
		t.Run(args.name, func(t *testing.T) {
			// Arrange
			player := lightshow.New(originalShow(t, 0), newCanvas())
			var log []string
			player.Log = func(message string) { log = append(log, message) }

			// Act
			for _, action := range args.actions {
				player.Action(action)
			}

			// Assert
			assert.Equal(t, args.log, log)
			assert.Equal(t, args.current, player.Current())
		})
	}

	t.Run("WhenKeyIsPressed", func(t *testing.T) {
		// Arrange
		player := lightshow.New(originalShow(t, 0), newCanvas())

		// Act
		player.Key(1)

		// Assert
		assert.Equal(t, 1, player.Current())
	})
}

func TestPlayerScript(t *testing.T) {
	// Arrange
	const (
		ldrLocalA  = 2 << 16
		pushA      = 20 << 16
		pop        = 19 << 16
		callSystem = 24 << 16
		retEvent   = 28 << 16
		clean      = 3
		wait       = 2
	)
	script := []uint32{
		4, 0, // init at word 4, no OnLoop event
		0x3F, 120, // resources
		ldrLocalA | 2, pushA, callSystem | clean, pop | 1,
		ldrLocalA | 3, pushA, callSystem | wait, pop | 1,
		retEvent,
	}
	b := compile(t, &blob.Show{Groups: [blob.GroupCount][]blob.CartoonData{{{Type: blob.Script, Script: script}}}})
	canvas := newCanvas()
	player := lightshow.New(lightshow.NewBlobShow(b, 0), canvas)

	// Act
	rendered, waitMs := player.Step(0)

	// Assert
	assert.True(t, rendered)
	assert.Equal(t, 120, waitMs)
	for _, voxel := range canvas.Displays[0].Voxels {
		require.Equal(t, byte(0x3F), voxel)
	}
}

func TestPlayerMissingResource(t *testing.T) {
	// Arrange
	b := compile(t, &blob.Show{Groups: [blob.GroupCount][]blob.CartoonData{{{
		Type:     blob.Normal,
		Header:   blob.FrameHeader{FrameCount: 1, Loop: 1, Delay: 10},
		Elements: []blob.ElementData{{Kind: blob.SetGraph, Graphs: []blob.GraphProp{{Graph: "nothing"}}}},
	}}}})
	player := lightshow.New(lightshow.NewBlobShow(b, 0), newCanvas())
	var log []string
	player.Log = func(message string) { log = append(log, message) }

	// Act
	player.Step(0)

	// Assert
	assert.Equal(t, []string{"Not found graph:'nothing'"}, log)
}
