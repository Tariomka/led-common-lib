package helpers

import (
	"reflect"
	"testing"
)

// TODO: make Parametrize take in structured test arguments instead of []any
// testArgs := []struct {
// 	y, z     uint8
// 	indexes  []uint8
// 	c        led.Color
// 	expected byte
// }{

func Parametrize[T any, V any](test *testing.T, testCase T, testArgs [][]V) {
	testCallback := reflect.ValueOf(testCase)
	for _, argSet := range testArgs {
		vargs := make([]reflect.Value, len(argSet)+1)

		vargs[0] = reflect.ValueOf(test)
		for i, arg := range argSet {
			vargs[i+1] = reflect.ValueOf(arg)
		}
		testCallback.Call(vargs)
	}
}
