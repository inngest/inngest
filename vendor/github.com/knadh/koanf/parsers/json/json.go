// Package json implements a koanf.Parser that parses JSON bytes as conf maps.
package json

import (
	"encoding/json"
)

// JSON implements a JSON parser.
type JSON struct{}

// Parser returns a JSON Parser.
func Parser() *JSON {
	return &JSON{}
}

// Unmarshal parses the given JSON bytes.
func (p *JSON) Unmarshal(b []byte) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Marshal marshals the given config map to JSON bytes.
func (p *JSON) Marshal(o map[string]interface{}) ([]byte, error) {
	return json.Marshal(o)
}
