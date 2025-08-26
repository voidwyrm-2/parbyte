package parbyte

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"unsafe"
)

type byteReader struct {
	inner      io.Reader
	amountRead uintptr
}

func (br *byteReader) notEnoughBytes(amount uintptr) error {
	return fmt.Errorf("Failed to read all %d bytes starting from position %x", amount, br.amountRead)
}

func (br *byteReader) read(amount uintptr) ([]byte, error) {
	buf := make([]byte, amount)

	read, err := br.inner.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	if uintptr(read) < amount {
		return nil, br.notEnoughBytes(amount)
	}

	br.amountRead += uintptr(read)

	return buf, err
}

// Unmarshal parses bytes of data and stores the result in the value pointed to by v.
// Unmarshal panics if v is nil or not a pointer.
//
// The binary formats of each Go value are as follows:
//
//   - bool, int8, uint8: byte
//   - int16, uint16: 2 bytes
//   - int32, uint32: 4 bytes
//   - int64, uint64: 8 bytes
//   - int, uint: 4 bytes on 32-bit systems, 8 bytes on 64-bit systems
//   - array with length N: N items are parsed recursively
//   - string: LEN bytes for N, then N bytes
//   - slice: LEN bytes for N, then N items are parsed recursively
//   - struct: same as slice, but the amount of items is determined by the amount of fields
//
// LEN is the amount of bytes specified by [Config.LenBytes].
func Unmarshal(data []byte, v any, config *Config) error {
	buf := bytes.NewBuffer(data)
	decoder := NewDecoder(buf, config)
	return decoder.Decode(v)
}

type Decoder struct {
	r byteReader
	c *Config
}

// NewDecoder returns a new decoder that reads from r.
//
// If config is nil, the default config will be used.
func NewDecoder(r io.Reader, config *Config) *Decoder {
	if config == nil {
		config = defaultConfig
	}

	return &Decoder{
		r: byteReader{r, 0},
		c: config,
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

	return d.decode(val)
}

func (d *Decoder) decode(v reflect.Value) (err error) {
	ty := v.Type()

	switch ty.Kind() {
	case reflect.Pointer, reflect.Interface:
		err = d.decode(v.Elem())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Float32, reflect.Float64, reflect.Complex64,
		reflect.Complex128, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64:
		err = d.decodeBytes(v, ty.Size())
	case reflect.String:
		err = d.decodeBytesWithLength(v)
	case reflect.Array:
		err = d.decodeSeq(v, ty.Size())
	case reflect.Slice:
		err = d.decodeSeqWithLength(v)
	// case reflect.Map:
	//	err = d.decodeMap(v)
	case reflect.Struct:
		err = d.decodeStruct(v)
	default:
		err = fmt.Errorf("Type %s (kind %s) cannot be unmarshaled into", ty, ty.Kind())
	}

	return
}

func (d *Decoder) decodeBytes(v reflect.Value, length uintptr) error {
	b, err := d.r.read(length)
	if err != nil {
		return err
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

func (d *Decoder) decodeBytesWithLength(v reflect.Value) error {
	length := uintptr(0)

	err := d.decodeBytes(reflect.ValueOf(&length), d.c.LenBytes)
	if err != nil {
		return err
	}

	return d.decodeBytes(v, length)
}

func (d *Decoder) decodeSeq(v reflect.Value, length uintptr) error {
	if v.Kind() == reflect.Slice {
		v.Grow(int(length))
		v.SetLen(int(length))
	}

	for i := range length {
		err := d.decode(v.Index(int(i)))
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) decodeSeqWithLength(v reflect.Value) error {
	length := uintptr(0)

	err := d.decodeBytes(reflect.ValueOf(&length), d.c.LenBytes)
	if err != nil {
		return err
	}

	return d.decodeSeq(v, length)
}

/*
func (d *Decoder) decodeMap(v reflect.Value) error {
	length := uintptr(0)

	err := d.decodeBytes(reflect.ValueOf(&length), d.c.LenFunc())
	if err != nil {
		return err
	}

	for i := range length {
		k := v.Type().Key()
	}

	return nil
}
*/

func (d *Decoder) decodeStruct(v reflect.Value) error {
	// v.Type().Field(i).Tag.Lookup()

	for i := range v.NumField() {
		fld := v.Field(i)
		err := d.decode(fld)
		if err != nil {
			return err
		}
	}

	return nil
}
