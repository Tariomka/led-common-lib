package uart_test

import (
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/communication/uart"
	"github.com/stretchr/testify/assert"
)

func TestParseLine(t *testing.T) {
	testArgs := []struct {
		line   string
		kind   uart.EventKind
		text   string
		values [5]int
	}{
		{"Mode=Text", uart.EventMode, "Text", [5]int{}},
		{"Change Cartoon 12", uart.EventCartoon, "", [5]int{12}},
		{"Next Cartoon", uart.EventNextCartoon, "", [5]int{}},
		{"Previous Cartoon", uart.EventPreviousCartoon, "", [5]int{}},
		{"Replay Cartoon", uart.EventReplayCartoon, "", [5]int{}},
		{"Add speed to 1.50", uart.EventSpeed, "", [5]int{150}},
		{"Sub speed to 0.10", uart.EventSpeed, "", [5]int{10}},
		{"Speed reset to 1.00", uart.EventSpeed, "", [5]int{100}},
		{"Play Model=0", uart.EventPlayModel, "", [5]int{0}},
		{"DeviceSuppend:1", uart.EventPause, "", [5]int{1}},
		{"Display off=1", uart.EventDisplayOff, "", [5]int{1}},
		{"DoKey5", uart.EventKey, "", [5]int{5}},
		{"IR:a8", uart.EventIR, "", [5]int{0xA8}},
		{"Show upload started", uart.EventUploadStarted, "", [5]int{}},
		{"Show upload OK", uart.EventUploadOK, "", [5]int{}},
		{"Show upload failed", uart.EventUploadFailed, "", [5]int{}},
		{"Led Cube is starting up", uart.EventStartup, "", [5]int{}},
		{"Watchdog reset!", uart.EventWatchdogReset, "", [5]int{}},
		{"T='8x32x8b'", uart.EventCubeType, "8x32x8b", [5]int{}},
		{"Config1=3,Config2=2", uart.EventConfig, "", [5]int{3, 2}},
		{"DeviceType=1,DeviceNum=0", uart.EventDevice, "", [5]int{1, 0}},
		{"Total=2", uart.EventTotal, "", [5]int{2}},
		{"Run x=-8,y=0,z=16,dir=1,type=7", uart.EventRun, "", [5]int{-8, 0, 16, 1, 7}},
		{"No uploaded show", uart.EventNoUploadedShow, "", [5]int{}},
		{"Uploaded show: 56 cartoons", uart.EventUploadedShow, "", [5]int{56}},
		{"Not Ready!", uart.EventNotReady, "", [5]int{}},
		{"Param Error!", uart.EventParamError, "", [5]int{}},
		{"Not found graph:'Fly_5'", uart.EventNotFound, "graph:'Fly_5'", [5]int{}},
		{"Warn: Pos > Cubes, 2,1", uart.EventWarning, "Pos > Cubes, 2,1", [5]int{}},
		{"Save settings error!", uart.EventError, "Save settings error!", [5]int{}},
		{"Regedit format error!", uart.EventError, "Regedit format error!", [5]int{}},
		{"Script error: division by zero", uart.EventError, "Script error: division by zero", [5]int{}},
		{"Commands explain:-----------------", uart.EventText, "", [5]int{}},
		{"Change Cartoon x", uart.EventText, "", [5]int{}},
		{"Run x=1,y=2", uart.EventText, "", [5]int{}},
		{"Config1=3,Config2=2,Config3=1", uart.EventText, "", [5]int{}},
		{"DoKey", uart.EventText, "", [5]int{}},
		{"", uart.EventText, "", [5]int{}},
	}

	for _, args := range testArgs {
		t.Run(args.line, func(t *testing.T) {
			// Arrange
			// Act
			actual := uart.ParseLine(args.line)

			// Assert
			assert.Equal(t, args.kind, actual.Kind)
			assert.Equal(t, args.line, actual.Line)
			assert.Equal(t, args.text, actual.Text)
			assert.Equal(t, args.values, actual.Values)
		})
	}
}

func TestParseStatus(t *testing.T) {
	t.Run("WhenLineIsValid", func(t *testing.T) {
		// Arrange
		line := "Status:Mode=Dynamic,Cartoon=3,Count=41,Speed=1.50,PlayModel=1,Pause=0,DisplayOff=1,DeviceNum=2"

		// Act
		actual := uart.ParseLine(line)

		// Assert
		assert.Equal(t, uart.EventStatus, actual.Kind)
		assert.Equal(t, uart.Status{
			Mode: "Dynamic", Cartoon: 3, Count: 41, Speed: 150, PlayAll: true, Pause: false, DisplayOff: true, DeviceNum: 2,
		}, actual.Status)
	})

	t.Run("WhenFieldIsUnknown", func(t *testing.T) {
		// Arrange
		// Act
		actual := uart.ParseLine("Status:Mode=Text,Count=0,Future=x")

		// Assert
		assert.Equal(t, uart.EventStatus, actual.Kind)
		assert.Equal(t, uart.Status{Mode: "Text"}, actual.Status)
	})

	t.Run("WhenValueIsInvalid", func(t *testing.T) {
		// Arrange
		// Act
		actual := uart.ParseLine("Status:Mode=Text,Cartoon=x")

		// Assert
		assert.Equal(t, uart.EventText, actual.Kind)
		assert.Equal(t, uart.Status{}, actual.Status)
	})
}
