# parbyte

A library for parsing binary formats into Go values, based on encoding/json and BurntSushi/toml.

## Installation

`go get github.com/voidwyrm-2/parbyte`

## Example

```go
package main

import (
	"fmt"
	"os"

	"github.com/voidwyrm-2/parbyte"
)

type metadata struct {
	Name     string
	Checksum [9]byte
	Content  []string
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

	fmt.Println(md.Name)
	fmt.Println(md.Checksum)
	fmt.Println(md.Content)
}
```
