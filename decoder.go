package parbyte

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"slices"
	"unsafe"
)

type byteReader struct {
	inner      io.Reader
	amountRead uintptr
}

func (br *byteReader) notEnoughBytes(amount uintptr, t reflect.Type) error {
	return fmt.Errorf("Failed to read all %d bytes starting from position %x for type %s", amount, br.amountRead, t)
}

func (br *byteReader) read(amount uintptr, t reflect.Type) ([]byte, error) {
	buf := make([]byte, amount)

	read, err := br.inner.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	if uintptr(read) < amount {
		return nil, br.notEnoughBytes(amount, t)
	}

	br.amountRead += uintptr(read)

	return buf, err
}

// Unmarshal parses bytes of data and stores the result in the value pointed to by v.
// Unmarshal panics if v is nil or not a pointer.
//
// The binary formats of each Go value are as follows:
//
//   - bool, int8, uint8: 1 byte.
//   - int16, uint16: 2 bytes.
//   - int32, uint32: 4 bytes.
//   - int64, uint64: 8 bytes.
//   - int, uint: 4 bytes on 32-bit systems, 8 bytes on 64-bit systems.
//   - array with length N: N items are parsed recursively.
//   - string: LEN bytes for N, then N bytes.
//   - slice: LEN bytes for N, then N items are parsed recursively.
//   - struct: same as slice, but the amount of items is determined by the amount of fields.
//
// LEN is the amount of bytes specified by [Config.LenBytes].
//
// The struct tags and their uses:
//
//   - `length`: if the value is a number, then that will be used as the field value length; if the value is a dot-separated path,
//     it will try to find it in the fields declared before the current field and use that value as the field value length.
//   - `endian`: the endianness of the field value; panics if the value is not 'big' or 'little', defaults to 'little'.
func Unmarshal(data []byte, v any, config *Config) error {
	buf := bytes.NewBuffer(data)
	decoder := NewDecoder(buf, config)
	return decoder.Decode(v)
}

type Decoder struct {
	r          byteReader
	fieldSizes map[string]uintptr
	c          *Config
}

// NewDecoder returns a new decoder that reads from r.
//
// If config is nil, the default config will be used.
//
// The decoder will not read bytes past what is needed.
func NewDecoder(r io.Reader, config *Config) *Decoder {
	if config == nil {
		config = DefaultConfig
	}

	return &Decoder{
		r:          byteReader{r, 0},
		fieldSizes: map[string]uintptr{},
		c:          config,
	}
}

// Decode reads the next value from its input and stores it in the value pointed to by v.
//
// See the documentation for [Unmarshal] for details about the conversion of bytes into a Go value.
func (d *Decoder) Decode(v any) error {
	val := reflect.ValueOf(v)
	if v == nil || val.Kind() != reflect.Pointer {
		panic("Decoder.Decode(v any): v must be a non-nil pointer")
	}

	return d.decode(localConfig{fieldSizes: d.fieldSizes, c: d.c, path: "-"}, val)
}

func (d *Decoder) decode(lc localConfig, v reflect.Value) (err error) {
	ty := v.Type()

	defer lc.saveValue(v)

	switch ty.Kind() {
	case reflect.Pointer, reflect.Interface:
		err = d.decode(lc, v.Elem())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32,
		reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Bool:
		err = d.decodeBytes(lc, v, ty.Size())
	case reflect.String:
		err = d.decodeBytesWithLength(lc, v)
	case reflect.Array:
		err = d.decodeSeq(lc, v, ty.Size())
	case reflect.Slice:
		err = d.decodeSeqWithLength(lc, v)
	// case reflect.Map:
	//	err = d.decodeMap(v)
	case reflect.Struct:
		err = d.decodeStruct(lc, v)
	default:
		err = fmt.Errorf("Type %s (kind %s) cannot be unmarshaled into", ty, ty.Kind())
	}

	return
}

func (d *Decoder) decodeBytes(lc localConfig, v reflect.Value, length uintptr) error {
	b, err := d.r.read(length, v.Type())
	if err != nil {
		return err
	}

	if lc.endian == "big" {
		k := v.Kind()
		if k == reflect.Pointer {
			k = v.Elem().Kind()
		}

		if (v.Kind() >= reflect.Int && v.Kind() <= reflect.Int64) || (v.Kind() >= reflect.Uint && v.Kind() <= reflect.Uint64) || v.Kind() == reflect.Uintptr {
			slices.Reverse(b)
		}
	}

	if v.Kind() == reflect.String {
		v.SetString(string(b))
	} else if v.Kind() == reflect.Pointer && v.Elem().Kind() == reflect.String {
		v.Elem().SetString(string(b))
	} else {
		var p unsafe.Pointer
		if v.Kind() == reflect.Pointer || v.Kind() == reflect.UnsafePointer {
			p = v.UnsafePointer()
		} else {
			p = v.Addr().UnsafePointer()
		}

		bp := (*byte)(p)
		sl := unsafe.Slice(bp, length)

		copy(sl, b)
	}

	return nil
}

func (d *Decoder) decodeBytesWithLength(lc localConfig, v reflect.Value) error {
	length := lc.length

	if length <= 0 {
		err := d.decodeBytes(lc, reflect.ValueOf(&length), d.c.LenBytes)
		if err != nil {
			return err
		}
	}

	return d.decodeBytes(lc, v, length)
}

func (d *Decoder) decodeSeq(lc localConfig, v reflect.Value, length uintptr) error {
	if v.Kind() == reflect.Slice {
		v.Grow(int(length))
		v.SetLen(int(length))
	}

	for i := range length {
		err := d.decode(lc.child(fmt.Sprint(i), nil), v.Index(int(i)))
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) decodeSeqWithLength(lc localConfig, v reflect.Value) error {
	length := lc.length

	if length <= 0 {
		err := d.decodeBytes(lc, reflect.ValueOf(&length), d.c.LenBytes)
		if err != nil {
			return err
		}
	}

	return d.decodeSeq(lc, v, length)
}

/*
func (d *Decoder) decodeMap(lc localConfig, v reflect.Value) error {
	length := lc.length

	if length <= 0 {
		err := d.decodeBytes(reflect.ValueOf(&length), d.c.LenFunc())
		if err != nil {
			return err
		}
	}

	for i := range length {
		k := v.Type().Key()
	}

	return nil
}
*/

func (d *Decoder) decodeStruct(lc localConfig, v reflect.Value) error {
	// v.Type().Field(i).Tag.Lookup()

	for i := range v.NumField() {
		fld := v.Field(i)
		fldty := v.Type().Field(i)

		err := d.decode(lc.child(fldty.Name, &fldty.Tag), fld)
		if err != nil {
			return err
		}
	}

	return nil
}
