package mocks

import (
	"github.com/stretchr/testify/mock"
)

type MockReadWriter struct {
	mock.Mock
}

func NewMockReadWriter() *MockReadWriter {
	mocked := new(MockReadWriter)
	return mocked
}

func (this *MockReadWriter) Read(buffer []byte) (int, error) {
	args := this.Called(buffer)
	return args.Int(0), args.Error(1)
}

func (this *MockReadWriter) Write(payload []byte) (int, error) {
	args := this.Called(payload)
	return args.Int(0), args.Error(1)
}
