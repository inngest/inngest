// Package yaml implements a koanf.Parser that parses YAML bytes as conf maps.
package yaml

import (
	"go.yaml.in/yaml/v3"
)

// YAML implements a YAML parser.
type YAML struct{}

// Parser returns a YAML Parser.
func Parser() *YAML {
	return &YAML{}
}

// Unmarshal parses the given YAML bytes.
func (p *YAML) Unmarshal(b []byte) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Marshal marshals the given config map to YAML bytes.
func (p *YAML) Marshal(o map[string]interface{}) ([]byte, error) {
	return yaml.Marshal(o)
}
