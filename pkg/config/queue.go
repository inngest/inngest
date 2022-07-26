package config

import (
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/config/registration"
)

type QueueService struct {
	Backend  string
	Concrete registration.QueueConfig
}

// UnmarshalJSON unmarshals the messaging service, keeping the raw bytes
// available for unmarshalling depending on the Backend type.
func (q *QueueService) UnmarshalJSON(byt []byte) error {
	type svc struct {
		Backend string
	}
	data := &svc{}
	if err := json.Unmarshal(byt, data); err != nil {
		return err
	}
	q.Backend = data.Backend

	f, ok := registration.RegisteredQueues()[q.Backend]
	if !ok {
		return fmt.Errorf("unknown queue backend: %s", q.Backend)
	}

	iface := f()
	if err := json.Unmarshal(byt, iface); err != nil {
		return err
	}
	q.Concrete = iface.(registration.QueueConfig)

	return nil
}
