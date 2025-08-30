package parbyte

import (
	"reflect"
	"strconv"
	"strings"
)

// Configuration for the Decoder, Encoder, Unmarshal, and Marshal.
type Config struct {
	// Store the values of fields integer fields.
	//
	// Defaults to true.
	StoreFieldValues bool
}

var DefaultConfig = &Config{
	StoreFieldValues: true,
}

type fieldValue struct {
	length   uintptr
	greedy   bool
	hasValue bool
}

type localConfig struct {
	fieldValues map[string]*fieldValue
	c           *Config
	path        string

	value         *fieldValue
	lengthSize    uintptr
	endian, flags string
}

func (lc *localConfig) child(fieldName string, tag *reflect.StructTag) *localConfig {
	c := &localConfig{
		fieldValues: lc.fieldValues,
		c:           lc.c,
		path:        lc.path,
		lengthSize:  4,
		endian:      "little",
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
		if length == "greedy:" {
			c.value = &fieldValue{greedy: true}
		} else {
			n, err := strconv.ParseUint(length, 0, 64)
			if err == nil {
				c.value = &fieldValue{length: uintptr(n)}
			} else {
				var ok bool
				lengthValue, ok := lc.fieldValues[length]
				if !ok {
					panic("'" + length + "' is not a known field")
				}

				c.value = lengthValue
			}

			c.value.hasValue = true
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

	lengthSize, ok := tag.Lookup("lengthSize")
	if ok {
		n, err := strconv.ParseUint(lengthSize, 0, 64)
		if err != nil {
			panic("Cannot parse '" + lengthSize + "' as an unsigned integer")
		} else if n == 0 {
			panic("'lengthSize' cannot be zero")
		}

		c.lengthSize = uintptr(n)
	}

	flags, ok := tag.Lookup("flags")
	if ok {
		c.flags = flags
	}

	return c
}

func (lc *localConfig) saveValue(v reflect.Value) {
	if !lc.c.StoreFieldValues {
		return
	}

	if v.Kind() == reflect.Pointer {
		lc.saveValue(v.Elem())
		return
	}

	if lc.path != "" {
		if v.Kind() == reflect.Uintptr {
			lc.fieldValues[lc.path] = &fieldValue{length: v.Interface().(uintptr)}
		} else if v.Kind() >= reflect.Uint && v.Kind() <= reflect.Uint64 {
			lc.fieldValues[lc.path] = &fieldValue{length: uintptr(v.Uint())}
		}
	}
}

func (lc *localConfig) hasFlag(name string) bool {
	if len(name) == 0 {
		return false
	}

	s := lc.flags
	for s != "" {
		var opt string
		opt, s, _ = strings.Cut(s, ",")
		if opt == name {
			return true
		}
	}

	return false
}
