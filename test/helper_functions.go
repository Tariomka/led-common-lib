package test

import "reflect"

func Parametrize[T any, V any](fn T, allValues [][]V) {
	v := reflect.ValueOf(fn)
	for _, a := range allValues {
		vargs := make([]reflect.Value, len(a))

		for i, b := range a {
			vargs[i] = reflect.ValueOf(b)
		}
		v.Call(vargs)
	}
}
