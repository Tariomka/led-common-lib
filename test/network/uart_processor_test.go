package network_test

import (
	"errors"
	"iter"
	"math/rand"
	"testing"

	"github.com/Tariomka/led-common-lib/pkg/network"
	"github.com/Tariomka/led-common-lib/test/helpers"
	"github.com/Tariomka/led-common-lib/test/helpers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUartProcessor_Read(t *testing.T) {
	t.Run("WhenHandshakeFailsBecauseOfWriteError", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(0, errors.New("failure"))
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actualDType, actualData, actualError := uartProcessor.Read()

		// Assert
		assert.Equal(t, network.UartEmpty, actualDType)
		assert.Nil(t, actualData)
		assert.Equal(t, network.ErrHandshake, actualError)
	})

	t.Run("WhenHandshakeFailsBecauseOfIncorrectWriteLength", func(t *testing.T) {
		testArgs := [][]any{
			{rand.Intn(12)},
			{rand.Int() + 13},
		}

		testCase := func(t *testing.T, writeLength int) {
			// Arrange
			mockUart := mocks.NewMockReadWriter()
			mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(writeLength, nil)
			uartProcessor := network.NewUartProcessor(mockUart)

			// Act
			actualDType, actualData, actualError := uartProcessor.Read()

			// Assert
			assert.Equal(t, network.UartEmpty, actualDType)
			assert.Nil(t, actualData)
			assert.Equal(t, network.ErrHandshake, actualError)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenHandshakeFailsBecauseOfReadError", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Return(0, errors.New("failure"))
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actualDType, actualData, actualError := uartProcessor.Read()

		// Assert
		assert.Equal(t, network.UartEmpty, actualDType)
		assert.Nil(t, actualData)
		assert.Equal(t, network.ErrHandshake, actualError)
	})

	t.Run("WhenHandshakeFailsBecauseOfWrongDataReceived", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("fake handshake"))
		}).Return(14, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actualDType, actualData, actualError := uartProcessor.Read()

		// Assert
		assert.Equal(t, network.UartEmpty, actualDType)
		assert.Nil(t, actualData)
		assert.Equal(t, network.ErrHandshake, actualError)
	})

	t.Run("WhenHandshakeIsReceivedInFragments", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFu"))
		}).Return(4, nil).Once()
		mockUart.On("Read", make([]byte, 1020)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("ZHNo"))
		}).Return(4, nil)
		mockUart.On("Read", make([]byte, 1016)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("YWtl"))
		}).Return(4, nil)
		mockUart.On("Read", make([]byte, 1024)).Return(0, nil)

		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actualDType, actualData, actualError := uartProcessor.Read()

		// Assert
		assert.Equal(t, network.UartEmpty, actualDType)
		assert.Nil(t, actualData)
		assert.Nil(t, actualError)
		mockUart.AssertNumberOfCalls(t, "Read", 4) // Handeshake in 3 packets + Read packet
	})

	t.Run("WhenHandshakeIsAlreadyEstablished", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil).Once()
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("\xAA\x00\x00\x00\x00\x00\x00\x00\x55"))
		}).Return(9, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		uartProcessor.Read()
		uartProcessor.Read()

		// Assert
		mockUart.AssertNumberOfCalls(t, "Read", 3) // Handshake + 2 Read packets
	})

	t.Run("WhenStartMarkerIsInvalid", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil).Once()
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("\x00"))
		}).Return(12, nil).Once()
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actualDType, actualData, actualError := uartProcessor.Read()

		// Assert
		assert.Equal(t, network.UartEmpty, actualDType)
		assert.Nil(t, actualData)
		assert.Equal(t, network.ErrExpectedStart, actualError)
	})

	t.Run("WhenContentLengthTooBig", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil).Once()
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("\xAA\x00\xFF\xFF"))
		}).Return(12, nil).Once()
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actualDType, actualData, actualError := uartProcessor.Read()

		// Assert
		assert.Equal(t, network.UartEmpty, actualDType)
		assert.Nil(t, actualData)
		assert.Equal(t, network.ErrTooMuchData, actualError)
	})

	t.Run("WhenEndMarkerIsInvalid", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil).Once()
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte{
				0xAA,       // Start marker
				0x00,       // UartEmpty data type
				0x00, 0x00, // Payload length (0 bytes)
				0x00, 0x00, 0x00, 0x00, // Checksum
				0xAA, // Invalid end marker
			})
		}).Return(12, nil).Once()
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actualDType, actualData, actualError := uartProcessor.Read()

		// Assert
		assert.Equal(t, network.UartEmpty, actualDType)
		assert.Nil(t, actualData)
		assert.Equal(t, network.ErrExpectedEnd, actualError)
	})

	t.Run("WhenChecksumIsInvalid", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil).Once()
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte{
				0xAA,       // Start marker
				0x00,       // UartEmpty data type
				0x00, 0x00, // Payload length (0 bytes)
				0xFF, 0xFF, 0xFF, 0xFF, // Invalid Checksum
				0x55, // End marker
			})
		}).Return(12, nil).Once()
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actualDType, actualData, actualError := uartProcessor.Read()

		// Assert
		assert.Equal(t, network.UartEmpty, actualDType)
		assert.Nil(t, actualData)
		assert.Equal(t, network.ErrChecksum, actualError)
	})

	t.Run("WhenPacketIsValid", func(t *testing.T) {
		testArgs := [][]any{
			{
				[]byte("\xAA\x00\x00\x03huh\xA8\xA4\x27\x53\x55"),
				network.UartEmpty,
				[]byte("huh"),
			},
			{
				[]byte("\xAA\x01\x00\x07Message\x79\x00\x09\xE3\x55"),
				network.UartMessage,
				[]byte("Message"),
			},
			{
				[]byte("\xAA\x02\x00\x04\xAA\xBB\xCC\xDD\x55\xB4\x01\xA7\x55"),
				network.UartBytes,
				[]byte("\xAA\xBB\xCC\xDD"),
			},
			{
				[]byte("\xAA\x03\x00\x0FSetting1=Value1\x55\x35\xEF\xDC\x55"),
				network.UartSetting,
				[]byte("Setting1=Value1"),
			},
			{[]byte("\xAA\x04\x00\x00\x00\x00\x00\x00\x55"), network.UartPing, []byte{}},
			{[]byte("\xAA\x05\x00\x00\x00\x00\x00\x00\x55"), network.UartPong, []byte{}},
			{
				[]byte("\xAA\xFF\x00\x10Fuck it we ball!\x2F\xC1\xE7\x60\x55"),
				network.UartDataType(0xFF),
				[]byte("Fuck it we ball!"),
			},
		}

		testCase := func(t *testing.T, readContent []byte, expectedDType network.UartDataType, expectedData []byte) {
			// Arrange
			mockUart := mocks.NewMockReadWriter()
			mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
			mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
				buf := args.Get(0).([]byte)
				copy(buf, []byte("SGFuZHNoYWtl"))
			}).Return(12, nil).Once()
			mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
				buf := args.Get(0).([]byte)
				copy(buf, readContent)
			}).Return(len(readContent), nil).Once()
			uartProcessor := network.NewUartProcessor(mockUart)

			// Act
			actualDType, actualData, actualError := uartProcessor.Read()

			// Assert
			assert.Equal(t, expectedDType, actualDType)
			assert.Equal(t, expectedData, actualData)
			assert.Nil(t, actualError)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenPacketIsReceivedInFragments", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil).Once()

		fullPacket := []byte("\xAA\x02\x00\x05\x01\x55\xAA\x55\x05\x64\xD8\xA4\x15\x55")
		for i, v := range fullPacket {
			mockUart.On("Read", make([]byte, 1024-i)).Run(func(args mock.Arguments) {
				buf := args.Get(0).([]byte)
				copy(buf, []byte{v})
			}).Return(1, nil).Once()
		}
		uartProcessor := network.NewUartProcessor(mockUart)

		expectedData := []byte("\x01\x55\xAA\x55\x05")

		// Act
		actualDType, actualData, actualError := uartProcessor.Read()

		// Assert
		assert.Equal(t, network.UartBytes, actualDType)
		assert.Equal(t, expectedData, actualData)
		assert.Nil(t, actualError)
	})

	t.Run("WhenMultiplePacketsAreReceivedInFragments", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil).Once()

		fragmentPacket := func(packet []byte) iter.Seq2[int, []byte] {
			return func(yield func(int, []byte) bool) {
				step := 3
				for i := 0; i < len(packet); i += step {
					if i+step > len(packet) {
						step = len(packet) - i
					}

					if !yield(i, packet[i:i+step]) {
						return
					}
				}
			}
		}
		firstPacketPart := []byte(
			"\xAA\x02\x00\x05\x01\x55\xAA\x55\x05\x64\xD8\xA4\x15\x55" +
				"\xAA") // Has training data of second packet
		secondPacketPart := []byte("\x02\x00\x07\x01\x55\xAA\x55\x05\x06\x07\x6F\xE9\xC5\xE7\x55")
		for offset, value := range fragmentPacket(firstPacketPart) {
			mockUart.On("Read", make([]byte, 1024-offset)).Run(func(args mock.Arguments) {
				buf := args.Get(0).([]byte)
				copy(buf, value)
			}).Return(len(value), nil).Once()
		}
		for offset, value := range fragmentPacket(secondPacketPart) {
			mockUart.On("Read", make([]byte, 1023-offset)).Run(func(args mock.Arguments) {
				buf := args.Get(0).([]byte)
				copy(buf, value)
			}).Return(len(value), nil).Once()
		}
		uartProcessor := network.NewUartProcessor(mockUart)

		expectedFirstData := []byte("\x01\x55\xAA\x55\x05")
		expectedSecondData := []byte("\x01\x55\xAA\x55\x05\x06\x07")

		// Act
		actualFirstDType, actualFirstData, actualFirstError := uartProcessor.Read()
		actualSecondDType, actualSecondData, actualSecondError := uartProcessor.Read()

		// Assert
		assert.Equal(t, network.UartBytes, actualFirstDType)
		assert.Equal(t, expectedFirstData, actualFirstData)
		assert.Nil(t, actualFirstError)

		assert.Equal(t, network.UartBytes, actualSecondDType)
		assert.Equal(t, expectedSecondData, actualSecondData)
		assert.Nil(t, actualSecondError)
	})

	t.Run("WhenMultiplePacketsAreReceivedInASingleFragment", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			// Hanshake and 2 packets received in a single fragment
			copy(buf, []byte(
				"SGFuZHNoYWtl"+
					"\xAA\x02\x00\x05\x01\x55\xAA\x55\x05\x64\xD8\xA4\x15\x55"+
					"\xAA\x02\x00\x07\x01\x55\xAA\x55\x05\x06\x07\x6F\xE9\xC5\xE7\x55"))
		}).Return(42, nil).Once()
		uartProcessor := network.NewUartProcessor(mockUart)

		expectedFirstData := []byte("\x01\x55\xAA\x55\x05")
		expectedSecondData := []byte("\x01\x55\xAA\x55\x05\x06\x07")

		// Act
		actualFirstDType, actualFirstData, actualFirstError := uartProcessor.Read()
		actualSecondDType, actualSecondData, actualSecondError := uartProcessor.Read()

		// Assert
		assert.Equal(t, network.UartBytes, actualFirstDType)
		assert.Equal(t, expectedFirstData, actualFirstData)
		assert.Nil(t, actualFirstError)

		assert.Equal(t, network.UartBytes, actualSecondDType)
		assert.Equal(t, expectedSecondData, actualSecondData)
		assert.Nil(t, actualSecondError)
	})
}

func TestUartProcessor_WriteMessage(t *testing.T) {
	t.Run("WhenHandshakeFailsBecauseOfWriteError", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(0, errors.New("failure"))
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.WriteMessage("Test message")

		// Assert
		assert.Equal(t, network.ErrHandshake, actual)
	})

	t.Run("WhenHandshakeFailsBecauseOfIncorrectWriteLength", func(t *testing.T) {
		testArgs := [][]any{
			{rand.Intn(12)},
			{rand.Int() + 13},
		}

		testCase := func(t *testing.T, writeLength int) {
			// Arrange
			mockUart := mocks.NewMockReadWriter()
			mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(writeLength, nil)
			uartProcessor := network.NewUartProcessor(mockUart)

			// Act
			actual := uartProcessor.WriteMessage("Test message")

			// Assert
			assert.Equal(t, network.ErrHandshake, actual)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenHandshakeFailsBecauseOfReadError", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Return(0, errors.New("failure"))
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.WriteMessage("Test message")

		// Assert
		assert.Equal(t, network.ErrHandshake, actual)
	})

	t.Run("WhenHandshakeFailsBecauseOfWrongDataReceived", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("fake handshake"))
		}).Return(14, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.WriteMessage("Test message")

		// Assert
		assert.Equal(t, network.ErrHandshake, actual)
	})

	t.Run("WhenHandshakeIsReceivedInFragments", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFu"))
		}).Return(4, nil).Once()
		mockUart.On("Read", make([]byte, 1020)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("ZHNo"))
		}).Return(4, nil)
		mockUart.On("Read", make([]byte, 1016)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("YWtl"))
		}).Return(4, nil)
		mockUart.On("Write", mock.Anything).Return(0, nil)

		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.WriteMessage("Test message")

		// Assert
		assert.Nil(t, actual)
		mockUart.AssertNumberOfCalls(t, "Read", 3)
	})

	t.Run("WhenHandshakeIsAlreadyEstablished", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil)
		mockUart.On("Write", mock.Anything).Return(0, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		expectedMessage1 := []byte{
			0xAA,       // Start marker
			0x01,       // UartMessage data type
			0x00, 0x08, // Payload length (8 bytes)
			'M', 'e', 's', 's', 'a', 'g', 'e', '1',
			0xBA, 0xA6, 0x5C, 0x7C, // Checksum
			0x55, // End marker
		}
		expectedMessage2 := []byte{
			0xAA,       // Start marker
			0x01,       // UartMessage data type
			0x00, 0x08, // Payload length (8 bytes)
			'M', 'e', 's', 's', 'a', 'g', 'e', '2',
			0x23, 0xAF, 0x0D, 0xC6, // Checksum
			0x55, // End marker
		}

		// Act
		uartProcessor.WriteMessage("Message1")
		actual := uartProcessor.WriteMessage("Message2")

		// Assert
		assert.Nil(t, actual)
		mockUart.AssertNumberOfCalls(t, "Write", 3) // Handshake + message1 + message2
		mockUart.AssertCalled(t, "Write", expectedMessage1)
		mockUart.AssertCalled(t, "Write", expectedMessage2)
		mockUart.AssertNumberOfCalls(t, "Read", 1) // Handshake
	})

	t.Run("WhenMessageFitsIntoASinglePacket", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil)
		mockUart.On("Write", mock.Anything).Return(0, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		expectedMessage := []byte{
			0xAA,       // Start marker
			0x01,       // UartMessage data type
			0x00, 0x0C, // Payload length (12 bytes)
			'T', 'e', 's', 't', ' ', 'm', 'e', 's', 's', 'a', 'g', 'e',
			0x07, 0xBD, 0xBC, 0x73, // Checksum
			0x55, // End marker
		}

		// Act
		actual := uartProcessor.WriteMessage("Test message")

		// Assert
		assert.Nil(t, actual)
		mockUart.AssertNumberOfCalls(t, "Write", 2) // Handshake + message
		mockUart.AssertCalled(t, "Write", expectedMessage)
		mockUart.AssertNumberOfCalls(t, "Read", 1) // Handshake
	})

	t.Run("WhenMessageDoesNotFitIntoASinglePacket", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil)
		mockUart.On("Write", mock.Anything).Return(0, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		expectedFirstPacket := []byte{
			0xAA,       // Start marker
			0x01,       // UartMessage data type
			0x03, 0xF7, // Payload length (1015 bytes)
			'L', 'o', 'r', 'e', 'm', ' ', 'i', 'p', 's', 'u', 'm', ' ', 'd', 'o', 'l', 'o', 'r', ' ',
			's', 'i', 't', ' ', 'a', 'm', 'e', 't', ',', ' ', 'c', 'o', 'n', 's', 'e', 'c', 't', 'e',
			't', 'u', 'r', ' ', 'a', 'd', 'i', 'p', 'i', 's', 'c', 'i', 'n', 'g', ' ', 'e', 'l', 'i',
			't', '.', ' ', 'P', 'e', 'l', 'l', 'e', 'n', 't', 'e', 's', 'q', 'u', 'e', ' ', 'i', 'd',
			' ', 'f', 'r', 'i', 'n', 'g', 'i', 'l', 'l', 'a', ' ', 'n', 'u', 'n', 'c', '.', '\n',
			'C', 'r', 'a', 's', ' ', 'e', 'g', 'e', 't', ' ', 'v', 'u', 'l', 'p', 'u', 't', 'a', 't',
			'e', ' ', 'e', 'x', '.', ' ', 'D', 'u', 'i', 's', ' ', 'v', 'e', 'h', 'i', 'c', 'u', 'l',
			'a', ' ', 'a', 'n', 't', 'e', ' ', 'n', 'i', 's', 'i', ',', ' ', 'a', 'l', 'i', 'q', 'u',
			'e', 't', ' ', 'c', 'o', 'm', 'm', 'o', 'd', 'o', ' ', 'd', 'o', 'l', 'o', 'r', ' ', 'f',
			'r', 'i', 'n', 'g', 'i', 'l', 'l', 'a', ' ', 'a', 't', '.', ' ', 'P', 'r', 'o', 'i', 'n',
			' ', 'p', 'o', 's', 'u', 'e', 'r', 'e', '\n', 'l', 'e', 'o', ' ', 'n', 'e', 'c', ' ',
			'm', 'i', ' ', 'i', 'a', 'c', 'u', 'l', 'i', 's', ',', ' ', 'a', 't', ' ', 'c', 'o', 'n',
			'd', 'i', 'm', 'e', 'n', 't', 'u', 'm', ' ', 'd', 'u', 'i', ' ', 'l', 'o', 'b', 'o', 'r',
			't', 'i', 's', '.', ' ', 'S', 'u', 's', 'p', 'e', 'n', 'd', 'i', 's', 's', 'e', ' ', 'm',
			'e', 't', 'u', 's', ' ', 'n', 'u', 'l', 'l', 'a', ',', ' ', 'o', 'r', 'n', 'a', 'r', 'e',
			' ', 'e', 'u', ' ', 'p', 'o', 's', 'u', 'e', 'r', 'e', ' ', 'e', 'g', 'e', 't', ',', '\n',
			'c', 'o', 'n', 'g', 'u', 'e', ' ', 'e', 'u', ' ', 'n', 'i', 'b', 'h', '.', ' ', 'P', 'h',
			'a', 's', 'e', 'l', 'l', 'u', 's', ' ', 'n', 'o', 'n', ' ', 'l', 'o', 'b', 'o', 'r', 't',
			'i', 's', ' ', 't', 'e', 'l', 'l', 'u', 's', '.', ' ', 'N', 'a', 'm', ' ', 'e', 'g', 'e',
			't', ' ', 'e', 'u', 'i', 's', 'm', 'o', 'd', ' ', 't', 'u', 'r', 'p', 'i', 's', '.', ' ',
			'A', 'e', 'n', 'e', 'a', 'n', ' ', 'a', 'n', 't', 'e', ' ', 'j', 'u', 's', 't', 'o', ',',
			' ', 'v', 'a', 'r', 'i', 'u', 's', '\n', 'e', 't', ' ', 'm', 'a', 's', 's', 'a', ' ',
			's', 'e', 'd', ',', ' ', 'v', 'u', 'l', 'p', 'u', 't', 'a', 't', 'e', ' ', 'd', 'i', 'c',
			't', 'u', 'm', ' ', 'm', 'a', 's', 's', 'a', '.', ' ', 'A', 'e', 'n', 'e', 'a', 'n', ' ',
			't', 'e', 'm', 'p', 'o', 'r', ' ', 'f', 'e', 'r', 'm', 'e', 'n', 't', 'u', 'm', ' ', 't',
			'u', 'r', 'p', 'i', 's', ',', ' ', 'a', ' ', 'p', 'r', 'e', 't', 'i', 'u', 'm', ' ', 'l',
			'o', 'r', 'e', 'm', '.', ' ', 'V', 'i', 'v', 'a', 'm', 'u', 's', ' ', 'e', 'g', 'e', 't',
			'\n', 'l', 'u', 'c', 't', 'u', 's', ' ', 'n', 'u', 'n', 'c', '.', ' ', 'P', 'r', 'a',
			'e', 's', 'e', 'n', 't', ' ', 'i', 'a', 'c', 'u', 'l', 'i', 's', ',', ' ', 'm', 'e', 't',
			'u', 's', ' ', 'e', 'g', 'e', 't', ' ', 'm', 'o', 'l', 'l', 'i', 's', ' ', 'u', 'l', 't',
			'r', 'i', 'c', 'e', 's', ',', ' ', 'e', 'n', 'i', 'm', ' ', 'm', 'e', 't', 'u', 's', ' ',
			'a', 'l', 'i', 'q', 'u', 'e', 't', ' ', 'e', 'x', ',', ' ', 'e', 't', ' ', 't', 'r', 'i',
			's', 't', 'i', 'q', 'u', 'e', ' ', 'd', 'o', 'l', 'o', 'r', '\n', 'n', 'i', 's', 'i',
			' ', 's', 'i', 't', ' ', 'a', 'm', 'e', 't', ' ', 'm', 'e', 't', 'u', 's', '.', ' ', 'M',
			'a', 'e', 'c', 'e', 'n', 'a', 's', ' ', 'b', 'l', 'a', 'n', 'd', 'i', 't', ',', ' ', 'n',
			'u', 'l', 'l', 'a', ' ', 'e', 'u', ' ', 'c', 'u', 'r', 's', 'u', 's', ' ', 's', 'o', 'd',
			'a', 'l', 'e', 's', ',', ' ', 's', 'e', 'm', ' ', 'p', 'u', 'r', 'u', 's', ' ', 'f', 'i',
			'n', 'i', 'b', 'u', 's', ' ', 's', 'a', 'p', 'i', 'e', 'n', ',', ' ', 'e', 'g', 'e', 't',
			' ', 'd', 'a', 'p', 'i', 'b', 'u', 's', '\n', 'a', 'r', 'c', 'u', ' ', 'l', 'e', 'o',
			' ', 'q', 'u', 'i', 's', ' ', 'l', 'e', 'o', '.', ' ', 'N', 'u', 'l', 'l', 'a', 'm', ' ',
			'c', 'u', 'r', 's', 'u', 's', ' ', 'j', 'u', 's', 't', 'o', ' ', 'a', 't', ' ', 'f', 'a',
			'c', 'i', 'l', 'i', 's', 'i', 's', ' ', 'p', 'o', 'r', 't', 'a', '.', ' ', 'I', 'n', ' ',
			'c', 'o', 'n', 'g', 'u', 'e', ' ', 'o', 'r', 'n', 'a', 'r', 'e', ' ', 'e', 'l', 'i', 't',
			' ', 'v', 'i', 't', 'a', 'e', ' ', 'f', 'r', 'i', 'n', 'g', 'i', 'l', 'l', 'a', '.', '\n',
			'Q', 'u', 'i', 's', 'q', 'u', 'e', ' ', 'f', 'e', 'r', 'm', 'e', 'n', 't', 'u', 'm', ' ',
			'e', 'g', 'e', 's', 't', 'a', 's', ' ', 'l', 'i', 'g', 'u', 'l', 'a', ',', ' ', 'd', 'i',
			'g', 'n', 'i', 's', 's', 'i', 'm', ' ', 'u', 'l', 't', 'r', 'i', 'c', 'e', 's', ' ', 'e',
			'l', 'i', 't', ' ', 'c', 'o', 'n', 'v', 'a', 'l', 'l', 'i', 's', ' ', 's', 'e', 'd', '.',
			' ', 'S', 'u', 's', 'p', 'e', 'n', 'd', 'i', 's', 's', 'e', ' ', 't', 'o', 'r', 't', 'o',
			'r', ' ', 'e', 'n', 'i', 'm', ',', '\n', 's', 'o', 'd', 'a', 'l', 'e', 's', ' ', 'i',
			'n', ' ', 'h', 'e', 'n', 'd', 'r', 'e', 'r', 'i', 't', ' ', 'a', 't', ',', ' ', 'm', 'o',
			'l', 'l', 'i', 's', ' ', 'i', 'n', ' ', 'o', 'd', 'i', 'o', '.', ' ', 'V', 'i', 'v', 'a',
			'm', 'u', 's', ' ', 'q', 'u', 'i', 's', ' ', 'e', 's', 't', ' ', 'a', 'c', ' ', 'o', 'd',
			'i', 'o', ' ', 't', 'e', 'm', 'p', 'u', 's', ' ', 's', 'e', 'm', 'p', 'e', 'r', ' ', 'g',
			'r', 'a', 'v', 'i', 'd', 'a', ' ', 'i', 'd', ' ', 'l', 'e', 'c', 't', 'u', 's', '.', '\n',
			'N', 'u', 'l', 'l', 'a', ' ', 'b', 'l', 'a', 'n', 'd', 'i', 't', ' ', 'u', 'r', 'n', 'a',
			' ', 'v', 'e', 'l', ' ', 'n', 'i', 's', 'l', ' ', 'p', '.', '.', '.',
			0xDD, 0x71, 0x1A, 0xBF, // Checksum
			0x55, // End marker
		}
		expectedSecondPacket := []byte{
			0xAA,       // Start marker
			0x01,       // UartMessage data type
			0x03, 0xE2, // Payload length (994 bytes)
			'.', '.', '.', 'h', 'a', 'r', 'e', 't', 'r', 'a', ',', ' ', 's', 'i', 't', ' ', 'a', 'm',
			'e', 't', ' ', 'v', 'e', 'n', 'e', 'n', 'a', 't', 'i', 's', ' ', 'q', 'u', 'a', 'm', ' ',
			'u', 'l', 'l', 'a', 'm', 'c', 'o', 'r', 'p', 'e', 'r', '.', ' ', 'I', 'n', 't', 'e', 'r',
			'd', 'u', 'm', ' ', 'e', 't', ' ', 'm', 'a', 'l', 'e', 's', 'u', 'a', 'd', 'a', ' ', 'f',
			'a', 'm', 'e', 's', '\n', 'a', 'c', ' ', 'a', 'n', 't', 'e', ' ', 'i', 'p', 's', 'u',
			'm', ' ', 'p', 'r', 'i', 'm', 'i', 's', ' ', 'i', 'n', ' ', 'f', 'a', 'u', 'c', 'i', 'b',
			'u', 's', '.', ' ', 'M', 'a', 'e', 'c', 'e', 'n', 'a', 's', ' ', 'u', 'l', 'l', 'a', 'm',
			'c', 'o', 'r', 'p', 'e', 'r', ' ', 'n', 'e', 'c', ' ', 'u', 'r', 'n', 'a', ' ', 'v', 'i',
			't', 'a', 'e', ' ', 'i', 'n', 't', 'e', 'r', 'd', 'u', 'm', '.', ' ', 'S', 'e', 'd', ' ',
			's', 'e', 'd', ' ', 'h', 'e', 'n', 'd', 'r', 'e', 'r', 'i', 't', '\n', 'l', 'i', 'g',
			'u', 'l', 'a', '.', ' ', 'N', 'u', 'l', 'l', 'a', ' ', 'f', 'a', 'c', 'i', 'l', 'i', 's',
			'i', '.', ' ', 'C', 'u', 'r', 'a', 'b', 'i', 't', 'u', 'r', ' ', 'e', 's', 't', ' ', 'o',
			'd', 'i', 'o', ',', ' ', 'f', 'i', 'n', 'i', 'b', 'u', 's', ' ', 's', 'i', 't', ' ', 'a',
			'm', 'e', 't', ' ', 'l', 'a', 'c', 'i', 'n', 'i', 'a', ' ', 'a', 'c', ',', ' ', 's', 'o',
			'l', 'l', 'i', 'c', 'i', 't', 'u', 'd', 'i', 'n', ' ', 'n', 'e', 'c', ' ', 'e', 'l', 'i',
			't', '.', '\n', 'A', 'e', 'n', 'e', 'a', 'n', ' ', 'u', 'l', 't', 'r', 'i', 'c', 'e',
			's', ',', ' ', 'd', 'u', 'i', ' ', 'u', 't', ' ', 'c', 'o', 'n', 'd', 'i', 'm', 'e', 'n',
			't', 'u', 'm', ' ', 'm', 'o', 'l', 'l', 'i', 's', ',', ' ', 'l', 'i', 'b', 'e', 'r', 'o',
			' ', 'e', 'n', 'i', 'm', ' ', 'c', 'o', 'n', 'v', 'a', 'l', 'l', 'i', 's', ' ', 'l', 'i',
			'g', 'u', 'l', 'a', ',', ' ', 'v', 'i', 't', 'a', 'e', ' ', 'a', 'u', 'c', 't', 'o', 'r',
			' ', 'n', 'i', 's', 'l', ' ', 'l', 'a', 'c', 'u', 's', ' ', 'i', 'd', '\n', 'j', 'u',
			's', 't', 'o', '.', ' ', 'I', 'n', ' ', 'i', 'a', 'c', 'u', 'l', 'i', 's', ' ', 'a', 'n',
			't', 'e', ' ', 'm', 'a', 's', 's', 'a', ',', ' ', 'u', 't', ' ', 'a', 'c', 'c', 'u', 'm',
			's', 'a', 'n', ' ', 'u', 'r', 'n', 'a', ' ', 'p', 'o', 's', 'u', 'e', 'r', 'e', ' ', 'n',
			'e', 'c', '.', ' ', 'V', 'e', 's', 't', 'i', 'b', 'u', 'l', 'u', 'm', ' ', 'i', 'd', ' ',
			'a', 'n', 't', 'e', ' ', 's', 'e', 'd', ' ', 't', 'u', 'r', 'p', 'i', 's', ' ', 'm', 'a',
			'x', 'i', 'm', 'u', 's', '\n', 'f', 'i', 'n', 'i', 'b', 'u', 's', '.', ' ', 'I', 'n',
			' ', 'h', 'a', 'c', ' ', 'h', 'a', 'b', 'i', 't', 'a', 's', 's', 'e', ' ', 'p', 'l', 'a',
			't', 'e', 'a', ' ', 'd', 'i', 'c', 't', 'u', 'm', 's', 't', '.', ' ', 'Q', 'u', 'i', 's',
			'q', 'u', 'e', ' ', 's', 'a', 'p', 'i', 'e', 'n', ' ', 'n', 'e', 'q', 'u', 'e', ',', ' ',
			'i', 'm', 'p', 'e', 'r', 'd', 'i', 'e', 't', ' ', 'n', 'o', 'n', ' ', 'f', 'i', 'n', 'i',
			'b', 'u', 's', ' ', 's', 'i', 't', ' ', 'a', 'm', 'e', 't', ',', '\n', 'p', 'h', 'a',
			'r', 'e', 't', 'r', 'a', ' ', 's', 'e', 'd', ' ', 's', 'e', 'm', '.', ' ', 'M', 'a', 'e',
			'c', 'e', 'n', 'a', 's', ' ', 'e', 'l', 'e', 'i', 'f', 'e', 'n', 'd', ' ', 'i', 'p', 's',
			'u', 'm', ' ', 's', 'i', 't', ' ', 'a', 'm', 'e', 't', ' ', 'm', 'e', 't', 'u', 's', ' ',
			'u', 'l', 't', 'r', 'i', 'c', 'i', 'e', 's', ',', ' ', 'n', 'e', 'c', ' ', 'v', 'i', 'v',
			'e', 'r', 'r', 'a', ' ', 'e', 'r', 'o', 's', ' ', 'f', 'e', 'u', 'g', 'i', 'a', 't', '.',
			'\n', 'M', 'a', 'e', 'c', 'e', 'n', 'a', 's', ' ', 's', 'o', 'l', 'l', 'i', 'c', 'i',
			't', 'u', 'd', 'i', 'n', ' ', 'a', 'r', 'c', 'u', ' ', 'i', 'n', ' ', 's', 'e', 'm', 'p',
			'e', 'r', ' ', 'c', 'o', 'n', 'd', 'i', 'm', 'e', 'n', 't', 'u', 'm', '.', ' ', 'C', 'r',
			'a', 's', ' ', 'm', 'a', 'l', 'e', 's', 'u', 'a', 'd', 'a', ',', ' ', 'e', 's', 't', ' ',
			'i', 'd', ' ', 'e', 'f', 'f', 'i', 'c', 'i', 't', 'u', 'r', ' ', 'v', 'u', 'l', 'p', 'u',
			't', 'a', 't', 'e', ',', ' ', 'e', 'r', 'a', 't', '\n', 'm', 'a', 'g', 'n', 'a', ' ',
			't', 'e', 'm', 'p', 'o', 'r', ' ', 'f', 'e', 'l', 'i', 's', ',', ' ', 'n', 'o', 'n', ' ',
			'v', 'i', 'v', 'e', 'r', 'r', 'a', ' ', 'n', 'e', 'q', 'u', 'e', ' ', 'q', 'u', 'a', 'm',
			' ', 'a', 'c', ' ', 'o', 'd', 'i', 'o', '.', ' ', 'F', 'u', 's', 'c', 'e', ' ', 'q', 'u',
			'i', 's', ' ', 'o', 'r', 'c', 'i', ' ', 'n', 'e', 'c', ' ', 'n', 'i', 's', 'l', ' ', 'r',
			'u', 't', 'r', 'u', 'm', ' ', 'g', 'r', 'a', 'v', 'i', 'd', 'a', '.', ' ', 'M', 'a', 'u',
			'r', 'i', 's', '\n', 'm', 'a', 'l', 'e', 's', 'u', 'a', 'd', 'a', ' ', 'e', 'r', 'o',
			's', ' ', 'a', 't', ' ', 'o', 'd', 'i', 'o', ' ', 'b', 'l', 'a', 'n', 'd', 'i', 't', ',',
			' ', 'n', 'e', 'c', ' ', 's', 'c', 'e', 'l', 'e', 'r', 'i', 's', 'q', 'u', 'e', ' ', 'n',
			'e', 'q', 'u', 'e', ' ', 'e', 'f', 'f', 'i', 'c', 'i', 't', 'u', 'r', '.', ' ', 'P', 'e',
			'l', 'l', 'e', 'n', 't', 'e', 's', 'q', 'u', 'e', ' ', 'h', 'a', 'b', 'i', 't', 'a', 'n',
			't', ' ', 'm', 'o', 'r', 'b', 'i', ' ', 't', 'r', 'i', 's', 't', 'i', 'q', 'u', 'e', '\n',
			's', 'e', 'n', 'e', 'c', 't', 'u', 's', ' ', 'e', 't', ' ', 'n', 'e', 't', 'u', 's', ' ',
			'e', 't', ' ', 'c', 'r', 'a', 's', ' ', 'a', 'm', 'e', 't', '.',
			0x22, 0x82, 0xB5, 0xE4, // Checksum
			0x55, // End marker
		}

		// Act
		actual := uartProcessor.WriteMessage(
			`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque id fringilla nunc.
Cras eget vulputate ex. Duis vehicula ante nisi, aliquet commodo dolor fringilla at. Proin posuere
leo nec mi iaculis, at condimentum dui lobortis. Suspendisse metus nulla, ornare eu posuere eget,
congue eu nibh. Phasellus non lobortis tellus. Nam eget euismod turpis. Aenean ante justo, varius
et massa sed, vulputate dictum massa. Aenean tempor fermentum turpis, a pretium lorem. Vivamus eget
luctus nunc. Praesent iaculis, metus eget mollis ultrices, enim metus aliquet ex, et tristique dolor
nisi sit amet metus. Maecenas blandit, nulla eu cursus sodales, sem purus finibus sapien, eget dapibus
arcu leo quis leo. Nullam cursus justo at facilisis porta. In congue ornare elit vitae fringilla.
Quisque fermentum egestas ligula, dignissim ultrices elit convallis sed. Suspendisse tortor enim,
sodales in hendrerit at, mollis in odio. Vivamus quis est ac odio tempus semper gravida id lectus.
Nulla blandit urna vel nisl pharetra, sit amet venenatis quam ullamcorper. Interdum et malesuada fames
ac ante ipsum primis in faucibus. Maecenas ullamcorper nec urna vitae interdum. Sed sed hendrerit
ligula. Nulla facilisi. Curabitur est odio, finibus sit amet lacinia ac, sollicitudin nec elit.
Aenean ultrices, dui ut condimentum mollis, libero enim convallis ligula, vitae auctor nisl lacus id
justo. In iaculis ante massa, ut accumsan urna posuere nec. Vestibulum id ante sed turpis maximus
finibus. In hac habitasse platea dictumst. Quisque sapien neque, imperdiet non finibus sit amet,
pharetra sed sem. Maecenas eleifend ipsum sit amet metus ultricies, nec viverra eros feugiat.
Maecenas sollicitudin arcu in semper condimentum. Cras malesuada, est id efficitur vulputate, erat
magna tempor felis, non viverra neque quam ac odio. Fusce quis orci nec nisl rutrum gravida. Mauris
malesuada eros at odio blandit, nec scelerisque neque efficitur. Pellentesque habitant morbi tristique
senectus et netus et cras amet.`)

		// Assert
		assert.Nil(t, actual)
		mockUart.AssertCalled(t, "Write", expectedFirstPacket)
		mockUart.AssertCalled(t, "Write", expectedSecondPacket)
	})
}

func TestUartProcessor_WriteBytes(t *testing.T) {
	t.Run("WhenHandshakeFailsBecauseOfWriteError", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(0, errors.New("failure"))
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.WriteBytes([]byte("Test message"))

		// Assert
		assert.Equal(t, network.ErrHandshake, actual)
	})

	t.Run("WhenHandshakeFailsBecauseOfIncorrectWriteLength", func(t *testing.T) {
		testArgs := [][]any{
			{rand.Intn(12)},
			{rand.Int() + 13},
		}

		testCase := func(t *testing.T, writeLength int) {
			// Arrange
			mockUart := mocks.NewMockReadWriter()
			mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(writeLength, nil)
			uartProcessor := network.NewUartProcessor(mockUart)

			// Act
			actual := uartProcessor.WriteBytes([]byte("Test message"))

			// Assert
			assert.Equal(t, network.ErrHandshake, actual)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenHandshakeFailsBecauseOfReadError", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Return(0, errors.New("failure"))
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.WriteBytes([]byte("Test message"))

		// Assert
		assert.Equal(t, network.ErrHandshake, actual)
	})

	t.Run("WhenHandshakeFailsBecauseOfWrongDataReceived", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("fake handshake"))
		}).Return(14, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.WriteBytes([]byte("Test message"))

		// Assert
		assert.Equal(t, network.ErrHandshake, actual)
	})

	t.Run("WhenHandshakeIsReceivedInFragments", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFu"))
		}).Return(4, nil).Once()
		mockUart.On("Read", make([]byte, 1020)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("ZHNo"))
		}).Return(4, nil)
		mockUart.On("Read", make([]byte, 1016)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("YWtl"))
		}).Return(4, nil)
		mockUart.On("Write", mock.Anything).Return(0, nil)

		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.WriteBytes([]byte("Test message"))

		// Assert
		assert.Nil(t, actual)
		mockUart.AssertNumberOfCalls(t, "Read", 3)
	})

	t.Run("WhenHandshakeIsAlreadyEstablished", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil)
		mockUart.On("Write", mock.Anything).Return(0, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		expectedMessage1 := []byte{
			0xAA,       // Start marker
			0x02,       // UartBytes data type
			0x00, 0x08, // Payload length (8 bytes)
			'M', 'e', 's', 's', 'a', 'g', 'e', '1',
			0xBA, 0xA6, 0x5C, 0x7C, // Checksum
			0x55, // End marker
		}
		expectedMessage2 := []byte{
			0xAA,       // Start marker
			0x02,       // UartBytes data type
			0x00, 0x08, // Payload length (8 bytes)
			'M', 'e', 's', 's', 'a', 'g', 'e', '2',
			0x23, 0xAF, 0x0D, 0xC6, // Checksum
			0x55, // End marker
		}

		// Act
		uartProcessor.WriteBytes([]byte("Message1"))
		actual := uartProcessor.WriteBytes([]byte("Message2"))

		// Assert
		assert.Nil(t, actual)
		mockUart.AssertNumberOfCalls(t, "Write", 3) // Handshake + message1 + message2
		mockUart.AssertCalled(t, "Write", expectedMessage1)
		mockUart.AssertCalled(t, "Write", expectedMessage2)
		mockUart.AssertNumberOfCalls(t, "Read", 1) //Handshake
	})

	t.Run("WhenMessageFitsIntoASinglePacket", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil)
		mockUart.On("Write", mock.Anything).Return(0, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		expectedMessage := []byte{
			0xAA,       // Start marker
			0x02,       // UartBytes data type
			0x00, 0x0C, // Payload length (12 bytes)
			'T', 'e', 's', 't', ' ', 'm', 'e', 's', 's', 'a', 'g', 'e',
			0x07, 0xBD, 0xBC, 0x73, // Checksum
			0x55, // End marker
		}

		// Act
		actual := uartProcessor.WriteBytes([]byte("Test message"))

		// Assert
		assert.Nil(t, actual)
		mockUart.AssertNumberOfCalls(t, "Write", 2) // Handshake + message
		mockUart.AssertCalled(t, "Write", expectedMessage)
		mockUart.AssertNumberOfCalls(t, "Read", 1) // Handshake
	})

	t.Run("WhenMessageDoesNotFitIntoASinglePacket", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil)
		mockUart.On("Write", mock.Anything).Return(0, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		expectedFirstPacket := []byte{
			0xAA,       // Start marker
			0x02,       // UartBytes data type
			0x03, 0xF7, // Payload length (1015 bytes)
			'L', 'o', 'r', 'e', 'm', ' ', 'i', 'p', 's', 'u', 'm', ' ', 'd', 'o', 'l', 'o', 'r', ' ',
			's', 'i', 't', ' ', 'a', 'm', 'e', 't', ',', ' ', 'c', 'o', 'n', 's', 'e', 'c', 't', 'e',
			't', 'u', 'r', ' ', 'a', 'd', 'i', 'p', 'i', 's', 'c', 'i', 'n', 'g', ' ', 'e', 'l', 'i',
			't', '.', ' ', 'P', 'e', 'l', 'l', 'e', 'n', 't', 'e', 's', 'q', 'u', 'e', ' ', 'i', 'd',
			' ', 'f', 'r', 'i', 'n', 'g', 'i', 'l', 'l', 'a', ' ', 'n', 'u', 'n', 'c', '.', '\n',
			'C', 'r', 'a', 's', ' ', 'e', 'g', 'e', 't', ' ', 'v', 'u', 'l', 'p', 'u', 't', 'a', 't',
			'e', ' ', 'e', 'x', '.', ' ', 'D', 'u', 'i', 's', ' ', 'v', 'e', 'h', 'i', 'c', 'u', 'l',
			'a', ' ', 'a', 'n', 't', 'e', ' ', 'n', 'i', 's', 'i', ',', ' ', 'a', 'l', 'i', 'q', 'u',
			'e', 't', ' ', 'c', 'o', 'm', 'm', 'o', 'd', 'o', ' ', 'd', 'o', 'l', 'o', 'r', ' ', 'f',
			'r', 'i', 'n', 'g', 'i', 'l', 'l', 'a', ' ', 'a', 't', '.', ' ', 'P', 'r', 'o', 'i', 'n',
			' ', 'p', 'o', 's', 'u', 'e', 'r', 'e', '\n', 'l', 'e', 'o', ' ', 'n', 'e', 'c', ' ',
			'm', 'i', ' ', 'i', 'a', 'c', 'u', 'l', 'i', 's', ',', ' ', 'a', 't', ' ', 'c', 'o', 'n',
			'd', 'i', 'm', 'e', 'n', 't', 'u', 'm', ' ', 'd', 'u', 'i', ' ', 'l', 'o', 'b', 'o', 'r',
			't', 'i', 's', '.', ' ', 'S', 'u', 's', 'p', 'e', 'n', 'd', 'i', 's', 's', 'e', ' ', 'm',
			'e', 't', 'u', 's', ' ', 'n', 'u', 'l', 'l', 'a', ',', ' ', 'o', 'r', 'n', 'a', 'r', 'e',
			' ', 'e', 'u', ' ', 'p', 'o', 's', 'u', 'e', 'r', 'e', ' ', 'e', 'g', 'e', 't', ',', '\n',
			'c', 'o', 'n', 'g', 'u', 'e', ' ', 'e', 'u', ' ', 'n', 'i', 'b', 'h', '.', ' ', 'P', 'h',
			'a', 's', 'e', 'l', 'l', 'u', 's', ' ', 'n', 'o', 'n', ' ', 'l', 'o', 'b', 'o', 'r', 't',
			'i', 's', ' ', 't', 'e', 'l', 'l', 'u', 's', '.', ' ', 'N', 'a', 'm', ' ', 'e', 'g', 'e',
			't', ' ', 'e', 'u', 'i', 's', 'm', 'o', 'd', ' ', 't', 'u', 'r', 'p', 'i', 's', '.', ' ',
			'A', 'e', 'n', 'e', 'a', 'n', ' ', 'a', 'n', 't', 'e', ' ', 'j', 'u', 's', 't', 'o', ',',
			' ', 'v', 'a', 'r', 'i', 'u', 's', '\n', 'e', 't', ' ', 'm', 'a', 's', 's', 'a', ' ',
			's', 'e', 'd', ',', ' ', 'v', 'u', 'l', 'p', 'u', 't', 'a', 't', 'e', ' ', 'd', 'i', 'c',
			't', 'u', 'm', ' ', 'm', 'a', 's', 's', 'a', '.', ' ', 'A', 'e', 'n', 'e', 'a', 'n', ' ',
			't', 'e', 'm', 'p', 'o', 'r', ' ', 'f', 'e', 'r', 'm', 'e', 'n', 't', 'u', 'm', ' ', 't',
			'u', 'r', 'p', 'i', 's', ',', ' ', 'a', ' ', 'p', 'r', 'e', 't', 'i', 'u', 'm', ' ', 'l',
			'o', 'r', 'e', 'm', '.', ' ', 'V', 'i', 'v', 'a', 'm', 'u', 's', ' ', 'e', 'g', 'e', 't',
			'\n', 'l', 'u', 'c', 't', 'u', 's', ' ', 'n', 'u', 'n', 'c', '.', ' ', 'P', 'r', 'a',
			'e', 's', 'e', 'n', 't', ' ', 'i', 'a', 'c', 'u', 'l', 'i', 's', ',', ' ', 'm', 'e', 't',
			'u', 's', ' ', 'e', 'g', 'e', 't', ' ', 'm', 'o', 'l', 'l', 'i', 's', ' ', 'u', 'l', 't',
			'r', 'i', 'c', 'e', 's', ',', ' ', 'e', 'n', 'i', 'm', ' ', 'm', 'e', 't', 'u', 's', ' ',
			'a', 'l', 'i', 'q', 'u', 'e', 't', ' ', 'e', 'x', ',', ' ', 'e', 't', ' ', 't', 'r', 'i',
			's', 't', 'i', 'q', 'u', 'e', ' ', 'd', 'o', 'l', 'o', 'r', '\n', 'n', 'i', 's', 'i',
			' ', 's', 'i', 't', ' ', 'a', 'm', 'e', 't', ' ', 'm', 'e', 't', 'u', 's', '.', ' ', 'M',
			'a', 'e', 'c', 'e', 'n', 'a', 's', ' ', 'b', 'l', 'a', 'n', 'd', 'i', 't', ',', ' ', 'n',
			'u', 'l', 'l', 'a', ' ', 'e', 'u', ' ', 'c', 'u', 'r', 's', 'u', 's', ' ', 's', 'o', 'd',
			'a', 'l', 'e', 's', ',', ' ', 's', 'e', 'm', ' ', 'p', 'u', 'r', 'u', 's', ' ', 'f', 'i',
			'n', 'i', 'b', 'u', 's', ' ', 's', 'a', 'p', 'i', 'e', 'n', ',', ' ', 'e', 'g', 'e', 't',
			' ', 'd', 'a', 'p', 'i', 'b', 'u', 's', '\n', 'a', 'r', 'c', 'u', ' ', 'l', 'e', 'o',
			' ', 'q', 'u', 'i', 's', ' ', 'l', 'e', 'o', '.', ' ', 'N', 'u', 'l', 'l', 'a', 'm', ' ',
			'c', 'u', 'r', 's', 'u', 's', ' ', 'j', 'u', 's', 't', 'o', ' ', 'a', 't', ' ', 'f', 'a',
			'c', 'i', 'l', 'i', 's', 'i', 's', ' ', 'p', 'o', 'r', 't', 'a', '.', ' ', 'I', 'n', ' ',
			'c', 'o', 'n', 'g', 'u', 'e', ' ', 'o', 'r', 'n', 'a', 'r', 'e', ' ', 'e', 'l', 'i', 't',
			' ', 'v', 'i', 't', 'a', 'e', ' ', 'f', 'r', 'i', 'n', 'g', 'i', 'l', 'l', 'a', '.', '\n',
			'Q', 'u', 'i', 's', 'q', 'u', 'e', ' ', 'f', 'e', 'r', 'm', 'e', 'n', 't', 'u', 'm', ' ',
			'e', 'g', 'e', 's', 't', 'a', 's', ' ', 'l', 'i', 'g', 'u', 'l', 'a', ',', ' ', 'd', 'i',
			'g', 'n', 'i', 's', 's', 'i', 'm', ' ', 'u', 'l', 't', 'r', 'i', 'c', 'e', 's', ' ', 'e',
			'l', 'i', 't', ' ', 'c', 'o', 'n', 'v', 'a', 'l', 'l', 'i', 's', ' ', 's', 'e', 'd', '.',
			' ', 'S', 'u', 's', 'p', 'e', 'n', 'd', 'i', 's', 's', 'e', ' ', 't', 'o', 'r', 't', 'o',
			'r', ' ', 'e', 'n', 'i', 'm', ',', '\n', 's', 'o', 'd', 'a', 'l', 'e', 's', ' ', 'i',
			'n', ' ', 'h', 'e', 'n', 'd', 'r', 'e', 'r', 'i', 't', ' ', 'a', 't', ',', ' ', 'm', 'o',
			'l', 'l', 'i', 's', ' ', 'i', 'n', ' ', 'o', 'd', 'i', 'o', '.', ' ', 'V', 'i', 'v', 'a',
			'm', 'u', 's', ' ', 'q', 'u', 'i', 's', ' ', 'e', 's', 't', ' ', 'a', 'c', ' ', 'o', 'd',
			'i', 'o', ' ', 't', 'e', 'm', 'p', 'u', 's', ' ', 's', 'e', 'm', 'p', 'e', 'r', ' ', 'g',
			'r', 'a', 'v', 'i', 'd', 'a', ' ', 'i', 'd', ' ', 'l', 'e', 'c', 't', 'u', 's', '.', '\n',
			'N', 'u', 'l', 'l', 'a', ' ', 'b', 'l', 'a', 'n', 'd', 'i', 't', ' ', 'u', 'r', 'n', 'a',
			' ', 'v', 'e', 'l', ' ', 'n', 'i', 's', 'l', ' ', 'p', 'h', 'a', 'r',
			0xBC, 0x5A, 0xA5, 0xD8, // Checksum
			0x55, // End marker
		}
		expectedSecondPacket := []byte{
			0xAA,       // Start marker
			0x02,       // UartBytes data type
			0x03, 0xDC, // Payload length (988 bytes)
			'e', 't', 'r', 'a', ',', ' ', 's', 'i', 't', ' ', 'a', 'm',
			'e', 't', ' ', 'v', 'e', 'n', 'e', 'n', 'a', 't', 'i', 's', ' ', 'q', 'u', 'a', 'm', ' ',
			'u', 'l', 'l', 'a', 'm', 'c', 'o', 'r', 'p', 'e', 'r', '.', ' ', 'I', 'n', 't', 'e', 'r',
			'd', 'u', 'm', ' ', 'e', 't', ' ', 'm', 'a', 'l', 'e', 's', 'u', 'a', 'd', 'a', ' ', 'f',
			'a', 'm', 'e', 's', '\n', 'a', 'c', ' ', 'a', 'n', 't', 'e', ' ', 'i', 'p', 's', 'u',
			'm', ' ', 'p', 'r', 'i', 'm', 'i', 's', ' ', 'i', 'n', ' ', 'f', 'a', 'u', 'c', 'i', 'b',
			'u', 's', '.', ' ', 'M', 'a', 'e', 'c', 'e', 'n', 'a', 's', ' ', 'u', 'l', 'l', 'a', 'm',
			'c', 'o', 'r', 'p', 'e', 'r', ' ', 'n', 'e', 'c', ' ', 'u', 'r', 'n', 'a', ' ', 'v', 'i',
			't', 'a', 'e', ' ', 'i', 'n', 't', 'e', 'r', 'd', 'u', 'm', '.', ' ', 'S', 'e', 'd', ' ',
			's', 'e', 'd', ' ', 'h', 'e', 'n', 'd', 'r', 'e', 'r', 'i', 't', '\n', 'l', 'i', 'g',
			'u', 'l', 'a', '.', ' ', 'N', 'u', 'l', 'l', 'a', ' ', 'f', 'a', 'c', 'i', 'l', 'i', 's',
			'i', '.', ' ', 'C', 'u', 'r', 'a', 'b', 'i', 't', 'u', 'r', ' ', 'e', 's', 't', ' ', 'o',
			'd', 'i', 'o', ',', ' ', 'f', 'i', 'n', 'i', 'b', 'u', 's', ' ', 's', 'i', 't', ' ', 'a',
			'm', 'e', 't', ' ', 'l', 'a', 'c', 'i', 'n', 'i', 'a', ' ', 'a', 'c', ',', ' ', 's', 'o',
			'l', 'l', 'i', 'c', 'i', 't', 'u', 'd', 'i', 'n', ' ', 'n', 'e', 'c', ' ', 'e', 'l', 'i',
			't', '.', '\n', 'A', 'e', 'n', 'e', 'a', 'n', ' ', 'u', 'l', 't', 'r', 'i', 'c', 'e',
			's', ',', ' ', 'd', 'u', 'i', ' ', 'u', 't', ' ', 'c', 'o', 'n', 'd', 'i', 'm', 'e', 'n',
			't', 'u', 'm', ' ', 'm', 'o', 'l', 'l', 'i', 's', ',', ' ', 'l', 'i', 'b', 'e', 'r', 'o',
			' ', 'e', 'n', 'i', 'm', ' ', 'c', 'o', 'n', 'v', 'a', 'l', 'l', 'i', 's', ' ', 'l', 'i',
			'g', 'u', 'l', 'a', ',', ' ', 'v', 'i', 't', 'a', 'e', ' ', 'a', 'u', 'c', 't', 'o', 'r',
			' ', 'n', 'i', 's', 'l', ' ', 'l', 'a', 'c', 'u', 's', ' ', 'i', 'd', '\n', 'j', 'u',
			's', 't', 'o', '.', ' ', 'I', 'n', ' ', 'i', 'a', 'c', 'u', 'l', 'i', 's', ' ', 'a', 'n',
			't', 'e', ' ', 'm', 'a', 's', 's', 'a', ',', ' ', 'u', 't', ' ', 'a', 'c', 'c', 'u', 'm',
			's', 'a', 'n', ' ', 'u', 'r', 'n', 'a', ' ', 'p', 'o', 's', 'u', 'e', 'r', 'e', ' ', 'n',
			'e', 'c', '.', ' ', 'V', 'e', 's', 't', 'i', 'b', 'u', 'l', 'u', 'm', ' ', 'i', 'd', ' ',
			'a', 'n', 't', 'e', ' ', 's', 'e', 'd', ' ', 't', 'u', 'r', 'p', 'i', 's', ' ', 'm', 'a',
			'x', 'i', 'm', 'u', 's', '\n', 'f', 'i', 'n', 'i', 'b', 'u', 's', '.', ' ', 'I', 'n',
			' ', 'h', 'a', 'c', ' ', 'h', 'a', 'b', 'i', 't', 'a', 's', 's', 'e', ' ', 'p', 'l', 'a',
			't', 'e', 'a', ' ', 'd', 'i', 'c', 't', 'u', 'm', 's', 't', '.', ' ', 'Q', 'u', 'i', 's',
			'q', 'u', 'e', ' ', 's', 'a', 'p', 'i', 'e', 'n', ' ', 'n', 'e', 'q', 'u', 'e', ',', ' ',
			'i', 'm', 'p', 'e', 'r', 'd', 'i', 'e', 't', ' ', 'n', 'o', 'n', ' ', 'f', 'i', 'n', 'i',
			'b', 'u', 's', ' ', 's', 'i', 't', ' ', 'a', 'm', 'e', 't', ',', '\n', 'p', 'h', 'a',
			'r', 'e', 't', 'r', 'a', ' ', 's', 'e', 'd', ' ', 's', 'e', 'm', '.', ' ', 'M', 'a', 'e',
			'c', 'e', 'n', 'a', 's', ' ', 'e', 'l', 'e', 'i', 'f', 'e', 'n', 'd', ' ', 'i', 'p', 's',
			'u', 'm', ' ', 's', 'i', 't', ' ', 'a', 'm', 'e', 't', ' ', 'm', 'e', 't', 'u', 's', ' ',
			'u', 'l', 't', 'r', 'i', 'c', 'i', 'e', 's', ',', ' ', 'n', 'e', 'c', ' ', 'v', 'i', 'v',
			'e', 'r', 'r', 'a', ' ', 'e', 'r', 'o', 's', ' ', 'f', 'e', 'u', 'g', 'i', 'a', 't', '.',
			'\n', 'M', 'a', 'e', 'c', 'e', 'n', 'a', 's', ' ', 's', 'o', 'l', 'l', 'i', 'c', 'i',
			't', 'u', 'd', 'i', 'n', ' ', 'a', 'r', 'c', 'u', ' ', 'i', 'n', ' ', 's', 'e', 'm', 'p',
			'e', 'r', ' ', 'c', 'o', 'n', 'd', 'i', 'm', 'e', 'n', 't', 'u', 'm', '.', ' ', 'C', 'r',
			'a', 's', ' ', 'm', 'a', 'l', 'e', 's', 'u', 'a', 'd', 'a', ',', ' ', 'e', 's', 't', ' ',
			'i', 'd', ' ', 'e', 'f', 'f', 'i', 'c', 'i', 't', 'u', 'r', ' ', 'v', 'u', 'l', 'p', 'u',
			't', 'a', 't', 'e', ',', ' ', 'e', 'r', 'a', 't', '\n', 'm', 'a', 'g', 'n', 'a', ' ',
			't', 'e', 'm', 'p', 'o', 'r', ' ', 'f', 'e', 'l', 'i', 's', ',', ' ', 'n', 'o', 'n', ' ',
			'v', 'i', 'v', 'e', 'r', 'r', 'a', ' ', 'n', 'e', 'q', 'u', 'e', ' ', 'q', 'u', 'a', 'm',
			' ', 'a', 'c', ' ', 'o', 'd', 'i', 'o', '.', ' ', 'F', 'u', 's', 'c', 'e', ' ', 'q', 'u',
			'i', 's', ' ', 'o', 'r', 'c', 'i', ' ', 'n', 'e', 'c', ' ', 'n', 'i', 's', 'l', ' ', 'r',
			'u', 't', 'r', 'u', 'm', ' ', 'g', 'r', 'a', 'v', 'i', 'd', 'a', '.', ' ', 'M', 'a', 'u',
			'r', 'i', 's', '\n', 'm', 'a', 'l', 'e', 's', 'u', 'a', 'd', 'a', ' ', 'e', 'r', 'o',
			's', ' ', 'a', 't', ' ', 'o', 'd', 'i', 'o', ' ', 'b', 'l', 'a', 'n', 'd', 'i', 't', ',',
			' ', 'n', 'e', 'c', ' ', 's', 'c', 'e', 'l', 'e', 'r', 'i', 's', 'q', 'u', 'e', ' ', 'n',
			'e', 'q', 'u', 'e', ' ', 'e', 'f', 'f', 'i', 'c', 'i', 't', 'u', 'r', '.', ' ', 'P', 'e',
			'l', 'l', 'e', 'n', 't', 'e', 's', 'q', 'u', 'e', ' ', 'h', 'a', 'b', 'i', 't', 'a', 'n',
			't', ' ', 'm', 'o', 'r', 'b', 'i', ' ', 't', 'r', 'i', 's', 't', 'i', 'q', 'u', 'e', '\n',
			's', 'e', 'n', 'e', 'c', 't', 'u', 's', ' ', 'e', 't', ' ', 'n', 'e', 't', 'u', 's', ' ',
			'e', 't', ' ', 'c', 'r', 'a', 's', ' ', 'a', 'm', 'e', 't', '.',
			0xDC, 0x95, 0x34, 0x2D, // Checksum
			0x55, // End marker
		}

		// Act
		actual := uartProcessor.WriteBytes([]byte(
			`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque id fringilla nunc.
Cras eget vulputate ex. Duis vehicula ante nisi, aliquet commodo dolor fringilla at. Proin posuere
leo nec mi iaculis, at condimentum dui lobortis. Suspendisse metus nulla, ornare eu posuere eget,
congue eu nibh. Phasellus non lobortis tellus. Nam eget euismod turpis. Aenean ante justo, varius
et massa sed, vulputate dictum massa. Aenean tempor fermentum turpis, a pretium lorem. Vivamus eget
luctus nunc. Praesent iaculis, metus eget mollis ultrices, enim metus aliquet ex, et tristique dolor
nisi sit amet metus. Maecenas blandit, nulla eu cursus sodales, sem purus finibus sapien, eget dapibus
arcu leo quis leo. Nullam cursus justo at facilisis porta. In congue ornare elit vitae fringilla.
Quisque fermentum egestas ligula, dignissim ultrices elit convallis sed. Suspendisse tortor enim,
sodales in hendrerit at, mollis in odio. Vivamus quis est ac odio tempus semper gravida id lectus.
Nulla blandit urna vel nisl pharetra, sit amet venenatis quam ullamcorper. Interdum et malesuada fames
ac ante ipsum primis in faucibus. Maecenas ullamcorper nec urna vitae interdum. Sed sed hendrerit
ligula. Nulla facilisi. Curabitur est odio, finibus sit amet lacinia ac, sollicitudin nec elit.
Aenean ultrices, dui ut condimentum mollis, libero enim convallis ligula, vitae auctor nisl lacus id
justo. In iaculis ante massa, ut accumsan urna posuere nec. Vestibulum id ante sed turpis maximus
finibus. In hac habitasse platea dictumst. Quisque sapien neque, imperdiet non finibus sit amet,
pharetra sed sem. Maecenas eleifend ipsum sit amet metus ultricies, nec viverra eros feugiat.
Maecenas sollicitudin arcu in semper condimentum. Cras malesuada, est id efficitur vulputate, erat
magna tempor felis, non viverra neque quam ac odio. Fusce quis orci nec nisl rutrum gravida. Mauris
malesuada eros at odio blandit, nec scelerisque neque efficitur. Pellentesque habitant morbi tristique
senectus et netus et cras amet.`))

		// Assert
		assert.Nil(t, actual)
		mockUart.AssertCalled(t, "Write", expectedFirstPacket)
		mockUart.AssertCalled(t, "Write", expectedSecondPacket)
	})
}

func TestUartProcessor_SendPing(t *testing.T) {
	// Arrange
	mockUart := mocks.NewMockReadWriter()
	mockUart.On("Write", mock.Anything).Return(0, nil)
	uartProcessor := network.NewUartProcessor(mockUart)

	expected := []byte{
		0xAA,       // Start marker
		0x04,       // UartPing data type
		0x00, 0x00, // Payload length (0 bytes)
		0x00, 0x00, 0x00, 0x00, // Checksum (0 bytes)
		0x55, // End marker
	}

	// Act
	uartProcessor.SendPing()

	// Assert
	mockUart.AssertNumberOfCalls(t, "Write", 1)
	mockUart.AssertCalled(t, "Write", expected)
}

func TestUartProcessor_SendPong(t *testing.T) {
	// Arrange
	mockUart := mocks.NewMockReadWriter()
	mockUart.On("Write", mock.Anything).Return(0, nil)
	uartProcessor := network.NewUartProcessor(mockUart)

	expected := []byte{
		0xAA,       // Start marker
		0x05,       // UartPong data type
		0x00, 0x00, // Payload length (0 bytes)
		0x00, 0x00, 0x00, 0x00, // Checksum (0 bytes)
		0x55, // End marker
	}

	// Act
	uartProcessor.SendPong()

	// Assert
	mockUart.AssertNumberOfCalls(t, "Write", 1)
	mockUart.AssertCalled(t, "Write", expected)
}

func TestUartProcessor_Desynchronize(t *testing.T) {
	// Arrange
	mockUart := mocks.NewMockReadWriter()
	mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
	mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, []byte("SGFuZHNoYWtl"))
	}).Return(12, nil)
	uartProcessor := network.NewUartProcessor(mockUart)

	// Act
	actualFirst := uartProcessor.Synchronize()
	uartProcessor.Desynchronize()
	actualSecond := uartProcessor.Synchronize()

	// Assert
	assert.Nil(t, actualFirst)
	assert.Nil(t, actualSecond)
	mockUart.AssertNumberOfCalls(t, "Read", 2)
}

func TestUartProcessor_Synchronize(t *testing.T) {
	t.Run("WhenHandshakeFailsBecauseOfWriteError", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(0, errors.New("failure"))
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.Synchronize()

		// Assert
		assert.Equal(t, network.ErrHandshake, actual)
	})

	t.Run("WhenHandshakeFailsBecauseOfIncorrectWriteLength", func(t *testing.T) {
		testArgs := [][]any{
			{rand.Intn(12)},
			{rand.Int() + 13},
		}

		testCase := func(t *testing.T, writeLength int) {
			// Arrange
			mockUart := mocks.NewMockReadWriter()
			mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(writeLength, nil)
			uartProcessor := network.NewUartProcessor(mockUart)

			// Act
			actual := uartProcessor.Synchronize()

			// Assert
			assert.Equal(t, network.ErrHandshake, actual)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("WhenHandshakeFailsBecauseOfReadError", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Return(0, errors.New("failure"))
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.Synchronize()

		// Assert
		assert.Equal(t, network.ErrHandshake, actual)
	})

	t.Run("WhenHandshakeFailsBecauseOfWrongDataReceived", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("fake handshake"))
		}).Return(14, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.Synchronize()

		// Assert
		assert.Equal(t, network.ErrHandshake, actual)
	})

	t.Run("WhenHandshakeFailsBecauseOfTimeout", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Return(0, nil)
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.Synchronize()

		// Assert
		assert.Equal(t, network.ErrHandshake, actual)
	})

	t.Run("WhenHandshakeIsReceivedInFragments", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFu"))
		}).Return(4, nil).Once()
		mockUart.On("Read", make([]byte, 1020)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("ZHNo"))
		}).Return(4, nil)
		mockUart.On("Read", make([]byte, 1016)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("YWtl"))
		}).Return(4, nil)

		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actual := uartProcessor.Synchronize()

		// Assert
		assert.Nil(t, actual)
		mockUart.AssertNumberOfCalls(t, "Read", 3)
	})

	t.Run("WhenHandshakeIsAlreadyEstablished", func(t *testing.T) {
		// Arrange
		mockUart := mocks.NewMockReadWriter()
		mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
		mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
			buf := args.Get(0).([]byte)
			copy(buf, []byte("SGFuZHNoYWtl"))
		}).Return(12, nil).Once()
		uartProcessor := network.NewUartProcessor(mockUart)

		// Act
		actualFirst := uartProcessor.Synchronize()
		actualSecond := uartProcessor.Synchronize()

		// Assert
		assert.Nil(t, actualFirst)
		assert.Nil(t, actualSecond)
		mockUart.AssertNumberOfCalls(t, "Read", 1)
	})
}

func TestUartProcessor(t *testing.T) {
	t.Run("MemoryAllocaions", func(t *testing.T) {
		testArgs := [][]any{
			{BenchmarkUartProcessor_SynchronizedSinglePacketRead, int64(58), float64(0)},
			{BenchmarkUartProcessor_SynchronizedMultiPacketRead, int64(59), float64(0)},
			{BenchmarkUartProcessor_SinglePacketRead, int64(133), float64(0)},
			{BenchmarkUartProcessor_MultiPacketRead, int64(266), float64(0)},
			{BenchmarkUartProcessor_Handshake, int64(127), float64(0)},
			{BenchmarkUartProcessor_MultiframgentHandshake, int64(290), float64(10)},
		}

		testCase := func(t *testing.T, benchmark func(*testing.B), expected int64, allocationRange float64) {
			// Act
			actual := testing.Benchmark(benchmark)

			// Assert
			assert.InDelta(t, expected, actual.AllocsPerOp(), allocationRange)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})

	t.Run("AllocationSize", func(t *testing.T) {
		testArgs := [][]any{
			{BenchmarkUartProcessor_SynchronizedSinglePacketRead, int64(24650)},
			{BenchmarkUartProcessor_SynchronizedMultiPacketRead, int64(24700)},
			{BenchmarkUartProcessor_SinglePacketRead, int64(32500)},
			{BenchmarkUartProcessor_MultiPacketRead, int64(65500)},
			{BenchmarkUartProcessor_Handshake, int64(23000)},
			{BenchmarkUartProcessor_MultiframgentHandshake, int64(139000)},
		}

		testCase := func(t *testing.T, benchmark func(*testing.B), expected int64) {
			// Act
			actual := testing.Benchmark(benchmark)

			// Assert
			assert.Less(t, actual.AllocedBytesPerOp(), expected)
		}

		helpers.Parametrize(t, testCase, testArgs)
	})
}

func BenchmarkUartProcessor_SynchronizedSinglePacketRead(b *testing.B) {
	// Arrange
	mockUart := mocks.NewMockReadWriter()
	mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
	mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, []byte("SGFuZHNoYWtl"))
	}).Return(12, nil).Once()
	mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, "\xAA\x00\x00\x03huh\xA8\xA4\x27\x53\x55")
	}).Return(12, nil)

	uartProcessor := network.NewUartProcessor(mockUart)

	// Act
	for b.Loop() {
		uartProcessor.Read()
	}
	b.ReportAllocs()
}

func BenchmarkUartProcessor_SynchronizedMultiPacketRead(b *testing.B) {
	// Arrange
	mockUart := mocks.NewMockReadWriter()
	mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
	mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, []byte("SGFuZHNoYWtl"))
	}).Return(12, nil).Once()
	mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, []byte(
			"\xAA\x02\x00\x05\x01\x55\xAA\x55\x05\x64\xD8\xA4\x15\x55"+
				"\xAA\x02\x00\x07\x01\x55\xAA\x55\x05\x06\x07\x6F\xE9\xC5\xE7\x55"))
	}).Return(30, nil)
	uartProcessor := network.NewUartProcessor(mockUart)

	// Act
	for b.Loop() {
		uartProcessor.Read()
		uartProcessor.Read()
	}
	b.ReportAllocs()
}

func BenchmarkUartProcessor_SinglePacketRead(b *testing.B) {
	// Arrange
	mockUart := mocks.NewMockReadWriter()
	mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
	mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, []byte("SGFuZHNoYWtl"))
	}).Return(12, nil).Once()
	mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, "\xAA\x00\x00\x03huh\xA8\xA4\x27\x53\x55")
	}).Return(12, nil)

	uartProcessor := network.NewUartProcessor(mockUart)

	// Act
	for b.Loop() {
		uartProcessor.Desynchronize()
		uartProcessor.Read()
	}
	b.ReportAllocs()
}

func BenchmarkUartProcessor_MultiPacketRead(b *testing.B) {
	// Arrange
	mockUart := mocks.NewMockReadWriter()
	mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
	mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, []byte("SGFuZHNoYWtl"))
	}).Return(12, nil).Once()
	mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, []byte(
			"\xAA\x02\x00\x05\x01\x55\xAA\x55\x05\x64\xD8\xA4\x15\x55"+
				"\xAA\x02\x00\x07\x01\x55\xAA\x55\x05\x06\x07\x6F\xE9\xC5\xE7\x55"))
	}).Return(30, nil)
	uartProcessor := network.NewUartProcessor(mockUart)

	// Act
	for b.Loop() {
		uartProcessor.Desynchronize()
		uartProcessor.Read()
		uartProcessor.Read()
	}
	b.ReportAllocs()
}

func BenchmarkUartProcessor_Handshake(b *testing.B) {
	// Arrange
	mockUart := mocks.NewMockReadWriter()
	mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
	mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, []byte("SGFuZHNoYWtl"))
	}).Return(12, nil)
	uartProcessor := network.NewUartProcessor(mockUart)

	// Act
	for b.Loop() {
		uartProcessor.Desynchronize()
		uartProcessor.Synchronize()
	}
	b.ReportAllocs()
}

func BenchmarkUartProcessor_MultiframgentHandshake(b *testing.B) {
	// Arrange
	mockUart := mocks.NewMockReadWriter()
	mockUart.On("Write", []byte("SGFuZHNoYWtl")).Return(12, nil)
	mockUart.On("Read", make([]byte, 1024)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, []byte("SGFu"))
	}).Return(4, nil)
	mockUart.On("Read", make([]byte, 1020)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, []byte("ZHNo"))
	}).Return(4, nil)
	mockUart.On("Read", make([]byte, 1016)).Run(func(args mock.Arguments) {
		buf := args.Get(0).([]byte)
		copy(buf, []byte("YWtl"))
	}).Return(4, nil)

	uartProcessor := network.NewUartProcessor(mockUart)

	// Act
	for b.Loop() {
		uartProcessor.Desynchronize()
		uartProcessor.Synchronize()
	}
	b.ReportAllocs()
}
