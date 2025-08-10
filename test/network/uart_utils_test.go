package network_test

import (
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/network"
	"github.com/stretchr/testify/assert"
)

func TestToUartMesssage(t *testing.T) {
	// Arrange
	expected := []byte("\xAA\x01\x00\x0CTest message\x07\xBD\xBC\x73\x55")

	// Act
	actual := network.ToUartMessage("Test message")

	// Assert
	assert.Equal(t, expected, actual)
}
