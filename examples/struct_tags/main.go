package main

import (
	"fmt"
	"os"

	"github.com/voidwyrm-2/parbyte"
)

type metadata struct {
	NameLength uint8
	Name       string `length:"NameLength"`
	Data       string `lengthSize:"2" endian:"big"`
	Meta       struct {
		A string `length:"8"`
		B string `length:"8"`
		C string `length:"8"`
	}
}

func main() {
	fr, err := os.Open("test.bin")
	if err != nil {
		panic(err)
	}

	defer fr.Close()

	decoder := parbyte.NewDecoder(fr, nil)

	md := metadata{}

	err = decoder.Decode(&md)
	if err != nil {
		panic(err)
	}

	fmt.Println(md.NameLength)
	fmt.Println(md.Name)
	fmt.Println(md.Data)
	fmt.Println(md.Meta)
}
