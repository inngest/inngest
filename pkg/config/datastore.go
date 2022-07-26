package config

import (
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/config/registration"
)

type DataStoreService struct {
	Backend  string
	Concrete registration.DataStoreConfig
}

// UnmarshalJSON unmarshals the messaging service, keeping the raw bytes
// available for unmarshalling depending on the Backend type.
func (s *DataStoreService) UnmarshalJSON(byt []byte) error {
	type svc struct {
		Backend string
	}
	data := &svc{}
	if err := json.Unmarshal(byt, data); err != nil {
		return err
	}
	s.Backend = data.Backend

	f, ok := registration.RegisteredDataStores()[s.Backend]
	if !ok {
		return fmt.Errorf("unknown datastore backend: %s", s.Backend)
	}
	iface := f()
	if err := json.Unmarshal(byt, iface); err != nil {
		return err
	}
	s.Concrete = iface.(registration.DataStoreConfig)

	return nil
}
