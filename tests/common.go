package tests

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func isInteger(k reflect.Kind) bool {
	return (k >= reflect.Int && k <= reflect.Int64) || (k >= reflect.Uint && k <= reflect.Uint64) || k == reflect.Uintptr
}

func testEqual(t *testing.T, value, expected any) {
	if path, a, b, eq := checkEqual(value, expected); !eq {
		sb := &strings.Builder{}

		sb.WriteString("Expected ")

		fmt.Fprintf(sb, "%v ", b)

		if isInteger(b.Kind()) {
			fmt.Fprintf(sb, "(hex %x) ", b)
		} else if b.Kind() == reflect.Array || b.Kind() == reflect.Slice {
			fmt.Fprintf(sb, "(length %d) ", b.Len())
		}

		fmt.Fprintf(sb, "at path %s, but found ", path)

		fmt.Fprintf(sb, "%v ", a)

		if isInteger(a.Kind()) {
			fmt.Fprintf(sb, "(hex %x) ", a)
		} else if a.Kind() == reflect.Array || a.Kind() == reflect.Slice {
			fmt.Fprintf(sb, "(length %d) ", a.Len())
		}

		sb.WriteString("instead")

		t.Fatal("\n", sb.String())
	}
}

func checkEqual(a, b any) (string, reflect.Value, reflect.Value, bool) {
	return checkEqualChild("root", reflect.ValueOf(a), reflect.ValueOf(b))
}

func checkEqualChild(path string, a, b reflect.Value) (string, reflect.Value, reflect.Value, bool) {
	if a.Type() != b.Type() && path == "root" {
		panic("a and b must be the same type")
	}

	switch a.Kind() {
	case reflect.Struct:
		for i := range a.NumField() {
			path, a, b, ok := checkEqualChild(path+"."+a.Type().Field(i).Name, a.Field(i), b.Field(i))
			if !ok {
				return path, a, b, false
			}
		}
	case reflect.Array, reflect.Slice:
		if a.Len() != b.Len() {
			return path, a, b, false
		}

		for i := range a.Len() {
			path, a, b, ok := checkEqualChild(fmt.Sprintf("%s.%d", path, i), a.Index(i), b.Index(i))
			if !ok {
				return path, a, b, false
			}
		}
	default:
		if eq := a.Equal(b); !eq {
			return path, a, b, false
		}
	}

	return path, a, b, true
}

func formatBytes(sb *strings.Builder, bytes []byte) {
	i := 0
	for _, b := range bytes {
		if i == 0 {
			sb.WriteString("  ")
		} else {
			sb.WriteString(" ")
		}

		fmt.Fprintf(sb, "%02x", b)

		i++

		if i == 4 {
			sb.WriteString("\n  -----------\n")
			i = 0
		}
	}
}
