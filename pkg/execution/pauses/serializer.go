package pauses

import (
	"encoding/json"
	"fmt"
)

const (
	encodingJSON = 0x00
)

// Serialize serializes a single block.  This prefixes the returned bytes with the
// current block encoding (1 byte, eg. json or protobuf) and a byte of metadata (eg.
// JSON version).  The rest of the bytes are the serialized block data.
func Serialize(block *Block, encoding, metadata byte) ([]byte, error) {
	switch encoding {
	case encodingJSON:
		return serializeJSON(block, metadata)
	default:
		return nil, fmt.Errorf("unknown encoding: %d", encoding)
	}

}

func Deserialize(byt []byte) (*Block, error) {
	switch byt[0] {
	case encodingJSON:
		return deserializeJSON(byt[1:])
	default:
		return nil, fmt.Errorf("unknown encoding: %d", byt[0])
	}
}

func serializeJSON(block *Block, metadata byte) ([]byte, error) {
	byt, err := json.Marshal(block)
	if err != nil {
		return nil, err
	}
	return append([]byte{encodingJSON, metadata}, byt...), nil
}

func deserializeJSON(byt []byte) (*Block, error) {
	block := &Block{}
	// metadata is currently unused, so trim the first byte off of the json
	if err := json.Unmarshal(byt[1:], block); err != nil {
		return nil, fmt.Errorf("error deserializing json block: %w", err)
	}
	return block, nil
}
