package tests

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/voidwyrm-2/parbyte"
)

type VelvetOpcode uint16

const (
	VelvetNop VelvetOpcode = iota
	VelvetRet
	VelvetHalt
	VelvetCall
	VelvetPush
	VelvetPop
	VelvetDup
	VelvetSwap
	VelvetRot
	VelvetSet
	VelvetJump
)

func (op VelvetOpcode) String() string {
	switch op {
	case VelvetNop:
		return "VelvetNop"
	case VelvetRet:
		return "VelvetRet"
	case VelvetHalt:
		return "VelvetHalt"
	case VelvetCall:
		return "VelvetCall"
	case VelvetPush:
		return "VelvetPush"
	case VelvetPop:
		return "VelvetPop"
	case VelvetDup:
		return "VelvetDup"
	case VelvetSwap:
		return "VelvetSwap"
	case VelvetRot:
		return "VelvetRot"
	case VelvetSet:
		return "VelvetSet"
	case VelvetJump:
		return "VelvetJump"
	default:
		panic(fmt.Sprintf("Invalid value %d for VelvetOpcode", op))
	}
}

type Instruction struct {
	Opcode   VelvetOpcode `endian:"big"`
	Flag     byte
	Operands [4]byte
}

func (i Instruction) String() string {
	return fmt.Sprintf(
		"%s(b%b) [%d %d %d %d]",
		i.Opcode,
		i.Flag,
		i.Operands[0],
		i.Operands[1],
		i.Operands[2],
		i.Operands[3],
	)
}

type VelvetHeader struct {
	Sig           string `length:"17"`
	Flag          byte
	VariableCount uint16 `endian:"big"`
	DataAddr      uint32 `endian:"big"`
	EntryAddr     uint32 `endian:"big"`
	Reserved      [4]byte
}

func (vh VelvetHeader) String() string {
	return fmt.Sprintf(`Sig: '%s'
Flag: b%b
VariableCount: %d
DataAddr: %d
EntryAddr: %d
Reserved: %02x%02x%02x%02x`,
		vh.Sig,
		vh.Flag,
		vh.VariableCount,
		vh.DataAddr,
		vh.EntryAddr,
		vh.Reserved[0],
		vh.Reserved[1],
		vh.Reserved[2],
		vh.Reserved[3],
	)
}

type VelvetExecutable struct {
	Header       VelvetHeader
	Instructions []Instruction `length:"3"`
	Data         []byte        `length:"greedy:"`
}

func (ve VelvetExecutable) String() string {
	sb := &strings.Builder{}

	fmt.Fprintf(sb, "Header:\n  %s\n", strings.Join(strings.Split(ve.Header.String(), "\n"), "\n  "))

	sb.WriteString("Instructions:\n")
	for _, ins := range ve.Instructions {
		fmt.Fprintf(sb, "  %s\n", ins)
	}

	sb.WriteString("Data:\n")
	formatBytes(sb, ve.Data)

	return strings.TrimSpace(sb.String())
}

func TestVelvet(t *testing.T) {
	fr, err := os.Open("testdata/hello.cvelv")
	if err != nil {
		t.Error(err)
		return
	}

	defer fr.Close()

	decoder := parbyte.NewDecoder(fr, nil)

	ve := VelvetExecutable{}

	err = decoder.Decode(&ve)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("\n", ve)

	/*
		Header:
		  Sig: 'Velvet Scarlatina'
		  Flag: b0
		  VariableCount: 0
		  DataAddr: 53
		  EntryAddr: 0
		  Reserved: 00000000
		Instructions:
		  VelvetPush(b10) [0 0 0 12]
		  VelvetCall(b0) [0 12 0 7]
		  VelvetHalt(b0) [0 0 0 0]
		Data:
		  65 6c 6c 6f
		  -----------
		  20 74 68 65
		  -----------
		  72 65 2e 70
		  -----------
		  72 69 6e 74
		  -----------
		  6c 6e 00
	*/

	expected := VelvetExecutable{
		Header: VelvetHeader{
			Sig:           "Velvet Scarlatina",
			Flag:          0b0,
			VariableCount: 0,
			DataAddr:      53,
			EntryAddr:     0,
			Reserved:      [4]byte{0x0, 0x0, 0x0, 0x0},
		},
		Instructions: []Instruction{
			{VelvetPush, 0b10, [4]byte{0, 0, 0, 12}},
			{VelvetCall, 0b0, [4]byte{0, 12, 0, 7}},
			{VelvetHalt, 0b0, [4]byte{0, 0, 0, 0}},
		},
		Data: []byte{
			0x68, 0x65, 0x6c, 0x6c,
			0x6f, 0x20, 0x74, 0x68,
			0x65, 0x72, 0x65, 0x2e,
			0x70, 0x72, 0x69, 0x6e,
			0x74, 0x6c, 0x6e, 0x00,
		},
	}

	testEqual(t, ve, expected)
}
