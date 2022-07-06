package config

import (
	"encoding/json"
	"fmt"
)

const (
	MessagingInMemory = "inmemory"
	MessagingNATS     = "nats"
)

// TopicURLCreator creates pub/sub topic URLs for the given backend
// implementation.
type TopicURLCreator interface {
	Backend() string
	TopicURL(topic string) string
}

// MessagingService represents
type MessagingService struct {
	Backend string

	raw      json.RawMessage
	concrete TopicURLCreator
}

// Set allows users to manually override the concrete backing config,
// for use within test environments.
func (m *MessagingService) Set(to TopicURLCreator) {
	m.Backend = to.Backend()
	m.concrete = to
}

func (m MessagingService) TopicURL(topic string) string {
	if m.concrete == nil {
		return ""
	}
	return m.concrete.TopicURL(topic)
}

// UnmarshalJSON unmarshals the messaging service, keeping the raw bytes
// available for unmarshalling depending on the Backend type.
func (m *MessagingService) UnmarshalJSON(byt []byte) error {
	m.raw = byt
	type svc struct {
		Backend string
	}

	data := &svc{}
	if err := json.Unmarshal(byt, data); err != nil {
		return err
	}

	m.Backend = data.Backend

	var concrete TopicURLCreator
	switch m.Backend {
	case MessagingInMemory:
		concrete = &InMemoryMessaging{}
	case MessagingNATS:
		concrete = &NATSMessaging{}
	default:
		return fmt.Errorf("unknown messaging backend: %s", m.Backend)
	}

	if err := json.Unmarshal(byt, concrete); err != nil {
		return err
	}

	m.concrete = concrete
	return nil
}

// NATS returns NATSMessaging configuration if the given Backend is
// "nats".
func (m MessagingService) NATS() (*NATSMessaging, error) {
	if m.Backend != "nats" {
		return nil, fmt.Errorf("messaging service mismatch: request nats, got %s", m.Backend)
	}
	c, _ := m.concrete.(*NATSMessaging)
	return c, nil
}

// InMemory returns InMemoryMessaging config when the backend is "inmemory".
func (m MessagingService) InMemory() (*InMemoryMessaging, error) {
	if m.Backend != "inmemory" {
		return nil, fmt.Errorf("messaging service mismatch: request inmemory, got %s", m.Backend)
	}
	c, _ := m.concrete.(*InMemoryMessaging)
	return c, nil
}

type InMemoryMessaging struct {
	Topic string
}

func (i InMemoryMessaging) Backend() string {
	return MessagingInMemory
}

func (i InMemoryMessaging) TopicURL(topic string) string {
	return fmt.Sprintf("mem://%s", topic)
}

type NATSMessaging struct {
	Topic     string
	ServerURL string
}

func (n NATSMessaging) Backend() string {
	return MessagingNATS
}

func (n NATSMessaging) TopicURL(topic string) string {
	return ""
}
