package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type URLType string

const (
	MessagingInMemory  = "inmemory"
	MessagingNATS      = "nats"
	MessagingGCPPubSub = "gcp-pubsub"

	URLTypePublish   URLType = "publish"
	URLTypeSubscribe URLType = "subscribe"
)

// TopicURLCreator creates pub/sub topic URLs for the given backend
// implementation.
type TopicURLCreator interface {
	Backend() string
	TopicURL(topic string, typ URLType) string
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

func (m MessagingService) TopicURL(topic string, typ URLType) string {
	if m.concrete == nil {
		return ""
	}
	return m.concrete.TopicURL(topic, typ)
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
	case MessagingGCPPubSub:
		concrete = &GCPPubSubMessaging{}
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

// InMemoryMessaging configures the topic for use with an in-memory
// pubsub backend.
type InMemoryMessaging struct {
	Topic string
}

func (i InMemoryMessaging) Backend() string {
	return MessagingInMemory
}

func (i InMemoryMessaging) TopicURL(topic string, typ URLType) string {
	return fmt.Sprintf("mem://%s", topic)
}

// NATSMessaging configures the NATS server URL and topic for use with
// a NATS messaging backend.
type NATSMessaging struct {
	Topic     string
	ServerURL string
}

func (n NATSMessaging) Backend() string {
	return MessagingNATS
}

func (n NATSMessaging) TopicURL(topic string, typ URLType) string {
	// Unfortunately, NATS uses an environment variable to configure the
	// remote URL.  This is hacky, but we set the remote URL prior to
	// connecting to any queues here.
	os.Setenv("NATS_SERVER_URL", n.ServerURL)
	return fmt.Sprintf("nats://%s", topic)
}

type GCPPubSubMessaging struct {
	Project string
	Topic   string
}

func (g GCPPubSubMessaging) Backend() string {
	return MessagingGCPPubSub
}

func (g GCPPubSubMessaging) TopicURL(topic string, typ URLType) string {
	if typ == URLTypePublish {
		return fmt.Sprintf("gcppubsub://projects/%s/topics/%s", g.Project, g.Topic)
	}

	return fmt.Sprintf("gcppubsub://projects/%s/subscriptions/%s", g.Project, g.Topic)
}
