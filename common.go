package parbyte

// Configuration for the Decoder, Encoder, Unmarshal, and Marshal.
type Config struct {
	// The size, in bytes, of item lengths.
	// See the documentation for [Unmarshal] for information.
	LenBytes uintptr
}

var defaultConfig = &Config{
	LenBytes: 4,
}
