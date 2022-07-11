package config

import "encoding/json"

// DriverConfig is an interface used to determine driver config structs.
type DriverConfig interface {
	// RuntimeName returns the name of the runtime used within the
	// driver implemetation and step configuration.
	RuntimeName() string
}

type DockerDriver struct {
	Host *string
}

func (DockerDriver) RuntimeName() string { return "docker" }

type HTTPDriver struct{}

func (HTTPDriver) RuntimeName() string { return "http" }

type MockDriver struct{}

func (MockDriver) RuntimeName() string { return "mock" }

// unmarshalDriver is used to help unmarshal drivers into
// their concrete structs.
type unmarshalDriver struct {
	Name string
	Raw  json.RawMessage
}

func (u *unmarshalDriver) UnmarshalJSON(b []byte) error {
	type driver struct {
		Name string
	}
	d := &driver{}
	if err := json.Unmarshal(b, d); err != nil {
		return err
	}
	u.Name = d.Name
	u.Raw = b
	return nil
}
