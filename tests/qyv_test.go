package tests

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/voidwyrm-2/parbyte"
)

type QyvHeader struct {
	Sig          string `length:"3"`
	Flags        byte
	LinkageSize  uint32 `endian:"big"`
	FunctionSize uint32 `endian:"big"`
	TextSize     uint32 `endian:"big"`
	DataSize     uint32 `endian:"big"`
	EntryAddr    uint16 `endian:"big"`
}

func (vh QyvHeader) String() string {
	return fmt.Sprintf(`Sig: '%s'
Flag: b%b
LinkageSize: %d
FunctionSize: %d
TextSize: %d
DataSize: %d
EntryAddr: %d`,
		vh.Sig,
		vh.Flags,
		vh.LinkageSize,
		vh.FunctionSize,
		vh.TextSize,
		vh.DataSize,
		vh.EntryAddr,
	)
}

type QyvExecutable struct {
	Header   QyvHeader
	Linkage  []byte `length:"Header.LinkageSize"`
	Function []byte `length:"Header.FunctionSize"`
	Text     []byte `length:"Header.TextSize"`
	Data     []byte `length:"Header.DataSize"`
}

func (qe QyvExecutable) String() string {
	sb := &strings.Builder{}

	fmt.Fprintf(sb, "Header:\n  %s\n", strings.Join(strings.Split(qe.Header.String(), "\n"), "\n  "))

	sb.WriteString("Text:\n")

	formatBytes(sb, qe.Text)

	sb.WriteString("\nData:\n")

	formatBytes(sb, qe.Data)

	return strings.TrimSpace(sb.String())
}

func TestQyv(t *testing.T) {
	fr, err := os.Open("testdata/basic.qyv")
	if err != nil {
		t.Error(err)
		return
	}

	defer fr.Close()

	decoder := parbyte.NewDecoder(fr, nil)

	qe := QyvExecutable{}

	err = decoder.Decode(&qe)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("\n", qe)

	/*
	   # header
	   "QYV" # signature
	   0 # flags
	   0 0 0 0 # linkage size
	   0 0 0 0 # function size
	   0 0 0 18 # text size
	   0 0 0 0 # data size
	   0 0 # entry point

	   # linkage
	   # 0 10 1 "testDevice"

	   # text
	   6 0 2 1 # set 2, x1
	   7 1 3 # copy x1, x3
	   14 0 1 3 4 # iadd x1, x3, x4
	   6 0 0 0 # set 0, x0
	   3 0 # exit x0

	   # data
	   # 0 0 0 0 0 0 0 0 0 0 0
	*/

	expected := QyvExecutable{
		Header: QyvHeader{
			Sig:          "QYV",
			LinkageSize:  0,
			FunctionSize: 0,
			TextSize:     18,
			DataSize:     0,
		},
		Linkage:  []byte{},
		Function: []byte{},
		Text: []byte{
			6, 0, 2, 1,
			7, 1, 3,
			14, 0, 1, 3, 4,
			6, 0, 0, 0,
			3, 0,
		},
		Data: []byte{},
	}

	testEqual(t, qe, expected)
}
