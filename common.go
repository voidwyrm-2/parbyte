//

package parbyte

import (
	"reflect"
	"strconv"
	"strings"
)

// Configuration for the Decoder, Encoder, Unmarshal, and Marshal.
type Config struct {
	// The size, in bytes, of item lengths.
	// See the documentation for [Unmarshal] for information on type formats.
	//
	// Defaults to 4.
	LenBytes uintptr

	// Store the values of fields integer fields.
	//
	// Defaults to true.
	StoreFieldValues bool
}

var DefaultConfig = &Config{
	LenBytes:         4,
	StoreFieldValues: true,
}

type localConfig struct {
	fieldSizes map[string]uintptr
	c          *Config
	path       string

	length uintptr
	endian string
}

func (lc localConfig) child(fieldName string, tag *reflect.StructTag) localConfig {
	c := localConfig{
		fieldSizes: lc.fieldSizes,
		c:          lc.c,
		path:       lc.path,
		endian:     "little",
	}

	if c.path == "" || c.path == "-" {
		c.path = fieldName
	} else if fieldName != "" {
		c.path += "." + fieldName
	}

	if tag == nil {
		return c
	}

	length, ok := tag.Lookup("length")
	if ok {
		n, err := strconv.ParseUint(length, 0, 64)
		if err == nil {
			c.length = uintptr(n)
		} else {
			var ok bool
			c.length, ok = lc.fieldSizes[length]
			if !ok {
				panic("'" + length + "' is not a known field")
			}
		}

	}

	endian, ok := tag.Lookup("endian")
	if ok {
		switch e := strings.ToLower(endian); e {
		case "big", "little":
			c.endian = e
		default:
			panic("'" + endian + "' is not a valid endianness")
		}
	}

	return c
}

func (lc localConfig) saveValue(v reflect.Value) {
	if !lc.c.StoreFieldValues {
		return
	}

	if v.Kind() == reflect.Pointer {
		lc.saveValue(v.Elem())
		return
	}

	if lc.path != "" {
		if v.Kind() == reflect.Uintptr {
			lc.fieldSizes[lc.path] = v.Interface().(uintptr)
		} else if v.Kind() >= reflect.Uint && v.Kind() <= reflect.Uint64 {
			lc.fieldSizes[lc.path] = uintptr(v.Uint())
		}
	}
}
