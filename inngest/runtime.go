package inngest

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	RuntimeTypeDocker = "docker"
	RuntimeTypeHTTP   = "http"
)

type Runtime interface {
	RuntimeType() string
}

type RuntimeWrapper struct {
	Runtime
}

func (r RuntimeWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Runtime)
}

func (r *RuntimeWrapper) UnmarshalJSON(b []byte) error {
	// XXX: This is wasteful, as we decode the runtime twice.  We can implement a custom decoder
	// which decodes and fills in one pass.
	interim := map[string]interface{}{}
	if err := json.Unmarshal(b, &interim); err != nil {
		return err
	}
	typ, ok := interim["type"]
	if !ok {
		return errors.New("unknown type")
	}

	switch typ {
	case RuntimeTypeDocker:
		docker := RuntimeDocker{}
		if err := json.Unmarshal(b, &docker); err != nil {
			return err
		}
		r.Runtime = docker
		return nil
	case RuntimeTypeHTTP:
		rt := RuntimeHTTP{}
		if err := json.Unmarshal(b, &rt); err != nil {
			return err
		}
		r.Runtime = rt
		return nil
	default:
		return fmt.Errorf("unknown runtime type: %s", typ)
	}
}

type RuntimeDocker struct {
	Entrypoint []string `json:"entrypoint,omitempty"`
	Memory     *int     `json:"memory"`
}

// MarshalJSON implements the JSON marshal interface so that cue can format this
// correctly when serializing actions.
func (r RuntimeDocker) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"type": RuntimeTypeDocker,
	}
	if len(r.Entrypoint) > 0 {
		data["entrypoint"] = r.Entrypoint
	}
	return json.Marshal(data)
}

func (RuntimeDocker) RuntimeType() string {
	return RuntimeTypeDocker
}

type RuntimeHTTP struct {
	URL string `json:"url"`
}

// MarshalJSON implements the JSON marshal interface so that cue can format this
// correctly when serializing actions.
func (r RuntimeHTTP) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"type": RuntimeTypeHTTP,
		"url":  r.URL,
	}
	return json.Marshal(data)
}

func (RuntimeHTTP) RuntimeType() string {
	return RuntimeTypeHTTP
}
