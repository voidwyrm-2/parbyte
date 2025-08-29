package parbyte

import "reflect"

func isInteger(k reflect.Kind) bool {
	return (k >= reflect.Int && k <= reflect.Int64) || (k >= reflect.Uint && k <= reflect.Uint64) || k == reflect.Uintptr
}
