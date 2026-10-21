package uart_test

import (
	"strings"
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/common"
	"github.com/Tariomka/led-common-lib/pkg/communication/uart"
	"github.com/stretchr/testify/assert"
)

func TestGetCommand(t *testing.T) {
	t.Run("WhenLineIsValid", func(t *testing.T) {
		// Arrange
		// Act
		actual, err := uart.GetCommand("Status")

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "#?Status?!", string(actual))
	})

	for name, line := range map[string]string{
		"WhenLineIsEmpty":      "",
		"WhenLineIsTooLong":    strings.Repeat("x", uart.MaxCommandLine+1),
		"WhenLineHasQuestion":  "a?b",
		"WhenLineHasEndMarker": "a?!b",
	} {
		t.Run(name, func(t *testing.T) {
			// Arrange
			// Act
			_, err := uart.GetCommand(line)

			// Assert
			assert.ErrorIs(t, err, common.ErrCommand)
		})
	}
}

func TestGetCommandHelpers(t *testing.T) {
	// Arrange
	setReg, setRegErr := uart.GetCommandSetReg("name", "value")
	getReg, getRegErr := uart.GetCommandGetReg("name")
	delReg, delRegErr := uart.GetCommandDelReg("name")
	testArgs := []struct {
		name     string
		actual   []byte
		expected string
	}{
		{"Key", uart.GetCommandKey(3), "#?Key 3?!"},
		{"Action", uart.GetCommandAction(12), "#?Action 12?!"},
		{"Cartoon", uart.GetCommandCartoon(40), "#?Cartoon 40?!"},
		{"Status", uart.GetCommandStatus(), "#?Status?!"},
		{"Help", uart.GetCommandHelp(), "#?Help?!"},
		{"RegList", uart.GetCommandRegList(), "#?RegList?!"},
		{"ClrReg", uart.GetCommandClrReg(), "#?ClrReg?!"},
		{"SetReg", setReg, "#?SetReg name value?!"},
		{"GetReg", getReg, "#?GetReg name?!"},
		{"DelReg", delReg, "#?DelReg name?!"},
	}

	// Act
	// Assert
	assert.NoError(t, setRegErr)
	assert.NoError(t, getRegErr)
	assert.NoError(t, delRegErr)
	for _, args := range testArgs {
		assert.Equal(t, args.expected, string(args.actual), args.name)
	}
}

func TestRegistryCommandErrors(t *testing.T) {
	// Arrange
	// Act
	_, setKeyErr := uart.GetCommandSetReg("a b", "c")
	_, setValueErr := uart.GetCommandSetReg("a", "b c")
	_, getErr := uart.GetCommandGetReg("")
	_, delErr := uart.GetCommandDelReg("a?")

	// Assert
	assert.ErrorIs(t, setKeyErr, common.ErrCommand)
	assert.ErrorIs(t, setValueErr, common.ErrCommand)
	assert.ErrorIs(t, getErr, common.ErrCommand)
	assert.ErrorIs(t, delErr, common.ErrCommand)
}
