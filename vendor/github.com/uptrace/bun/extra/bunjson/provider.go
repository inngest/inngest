package bunjson

import (
	"io"
)

var provider Provider = StdProvider{}

func SetProvider(p Provider) {
	provider = p
}

type Provider interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
	NewEncoder(w io.Writer) Encoder
	NewDecoder(r io.Reader) Decoder
}

type Decoder interface {
	Decode(v any) error
	UseNumber()
}

type Encoder interface {
	Encode(v any) error
}

func Marshal(v any) ([]byte, error) {
	return provider.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	return provider.Unmarshal(data, v)
}

func NewEncoder(w io.Writer) Encoder {
	return provider.NewEncoder(w)
}

func NewDecoder(r io.Reader) Decoder {
	return provider.NewDecoder(r)
}
