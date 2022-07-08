package config

import (
	"encoding/json"
	"fmt"
)

type QueueService struct {
	Backend  string
	Concrete interface{}
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

	switch q.Backend {
	case "inmemory":
		q.Concrete = &InMemoryQueue{}
	default:
		return fmt.Errorf("unknown queue backend: %s", q.Backend)
	}

	return nil
}

type InMemoryQueue struct{}
