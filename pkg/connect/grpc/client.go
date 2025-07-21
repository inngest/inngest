package grpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/inngest/inngest/pkg/logger"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	grpcLib "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ErrGatewayNotFound = fmt.Errorf("gateway not found")

type PingableClient interface {
	Ping(ctx context.Context, req *connectpb.PingRequest, opts ...grpcLib.CallOption) (*connectpb.PingResponse, error)
}

type GRPCClientManager[T PingableClient] struct {
	mu          sync.RWMutex
	clients     map[string]T
	connections map[string]*grpcLib.ClientConn
	logger      logger.Logger
	factory     func(grpcLib.ClientConnInterface) T
	dialer      func(target string, opts ...grpcLib.DialOption) (*grpcLib.ClientConn, error)
}

type GRPCClientManagerOption[T PingableClient] func(*GRPCClientManager[T])

func WithDialer[T PingableClient](dialer func(target string, opts ...grpcLib.DialOption) (*grpcLib.ClientConn, error)) GRPCClientManagerOption[T] {
	return func(m *GRPCClientManager[T]) {
		m.dialer = dialer
	}
}

func WithLogger[T PingableClient](logger logger.Logger) GRPCClientManagerOption[T] {
	return func(m *GRPCClientManager[T]) {
		m.logger = logger
	}
}

func NewGRPCClientManager[T PingableClient](factory func(grpcLib.ClientConnInterface) T, opts ...GRPCClientManagerOption[T]) *GRPCClientManager[T] {
	m := &GRPCClientManager[T]{
		clients:     make(map[string]T),
		connections: make(map[string]*grpcLib.ClientConn),
		factory:     factory,
		dialer:      grpcLib.NewClient,
		logger:      logger.VoidLogger(),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

func (m *GRPCClientManager[T]) GetClient(ctx context.Context, key string) (T, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if client, exists := m.clients[key]; exists {
		return client, nil
	}
	var zero T
	return zero, ErrGatewayNotFound
}

func (m *GRPCClientManager[T]) GetOrCreateClient(ctx context.Context, key string, grpcURL string) (T, error) {
	m.mu.RLock()
	if client, exists := m.clients[key]; exists {
		m.mu.RUnlock()
		return client, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if client, exists := m.clients[key]; exists {
		return client, nil
	}

	m.logger.Info("grpc client not found, creating one dynamically", "key", key)

	conn, err := m.dialer(grpcURL, grpcLib.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		var zero T
		return zero, fmt.Errorf("could not create grpc client for %s: %w", grpcURL, err)
	}

	client := m.factory(conn)

	result, err := client.Ping(ctx, &connectpb.PingRequest{})
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			m.logger.Error("failed to close grpc connection after ping failure", "url", grpcURL, "error", closeErr)
		}
		var zero T
		return zero, fmt.Errorf("could not ping %s: %w", grpcURL, err)
	}

	if result.GetMessage() != "ok" {
		if closeErr := conn.Close(); closeErr != nil {
			m.logger.Error("failed to close grpc connection after unexpected ping response", "url", grpcURL, "error", closeErr)
		}
		var zero T
		return zero, fmt.Errorf("unexpected ping response from %s: %s", grpcURL, result.GetMessage())
	}

	m.clients[key] = client
	m.connections[key] = conn
	return client, nil
}

func (m *GRPCClientManager[T]) RemoveClient(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[key]; exists {
		if err := conn.Close(); err != nil {
			m.logger.Error("failed to close grpc connection", "url", key, "error", err)
		}
		delete(m.connections, key)
	}

	delete(m.clients, key)
}

func (m *GRPCClientManager[T]) GetClientKeys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.clients))
	for url := range m.clients {
		keys = append(keys, url)
	}
	return keys
}
