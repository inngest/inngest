package config

import (
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest-cli/pkg/config/registration"
)

type StateService struct {
	Backend  string
	Concrete registration.StateConfig
}

// UnmarshalJSON unmarshals the messaging service, keeping the raw bytes
// available for unmarshalling depending on the Backend type.
func (s *StateService) UnmarshalJSON(byt []byte) error {
	type svc struct {
		Backend string
	}
	data := &svc{}
	if err := json.Unmarshal(byt, data); err != nil {
		return err
	}
	s.Backend = data.Backend

	f, ok := registration.RegisteredStates()[s.Backend]
	if !ok {
		return fmt.Errorf("unknown state backend: %s", s.Backend)
	}
	iface := f()
	if err := json.Unmarshal(byt, iface); err != nil {
		return err
	}
	s.Concrete = iface.(registration.StateConfig)

	return nil
}
