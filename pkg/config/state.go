package config

import (
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type StateService struct {
	Backend  string
	Concrete interface{}
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

	switch s.Backend {
	case "inmemory":
		s.Concrete = &InMemoryState{}
	case "redis":
		s.Concrete = &RedisState{}
	default:
		return fmt.Errorf("unknown state backend: %s", s.Backend)
	}

	return json.Unmarshal(byt, s.Concrete)
}

type InMemoryState struct{}

type RedisState struct {
	Host       string
	Port       int
	DB         int
	Username   string
	Password   string
	MaxRetries *int
	PoolSize   *int
}

func (r *RedisState) ConnectOpts() redis.Options {
	opts := redis.Options{
		Addr:     fmt.Sprintf("%s:%d", r.Host, r.Port),
		DB:       r.DB,
		Username: r.Username,
		Password: r.Password,
	}

	if r.MaxRetries != nil {
		opts.MaxRetries = *r.MaxRetries
	}

	if r.PoolSize != nil {
		opts.PoolSize = *r.PoolSize
	}

	return opts
}
