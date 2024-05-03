package state

import (
	"context"
	"encoding/json"
)

// TODO: Create codecs for storing state easily.
type MetadataCodec interface {
	Encode(ctx context.Context, m Metadata) ([]byte, error)
	Decode(ctx context.Context, data []byte, m *Metadata) error
}

func JSONCodec() MetadataCodec {
	return jsoncodec{}
}

type jsoncodec struct{}

func (jsoncodec) Encode(ctx context.Context, m Metadata) ([]byte, error) {
	// TODO: Add JSON codec prefix
	return json.Marshal(m)
}

func (jsoncodec) Decode(ctx context.Context, data []byte, m *Metadata) error {
	// TODO: Check JSON codec prefix, remove
	return json.Unmarshal(data, m)
}
