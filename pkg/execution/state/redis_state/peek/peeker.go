package peek

import (
	"context"

	"github.com/google/uuid"
	"github.com/redis/rueidis"
)

type Result[T any] struct {
	Items        []*T
	TotalCount   int
	RemovedCount int
}

// Peeker defines the interface for peeking operations on queues
type Peeker[T any] interface {
	Peek(ctx context.Context, keyOrderedPointerSet string, opt ...Option) (*Result[T], error)
	PeekPointer(ctx context.Context, keyOrderedPointerSet string, opt ...Option) ([]string, error)
	PeekUUIDPointer(ctx context.Context, keyOrderedPointerSet string, opt ...Option) ([]uuid.UUID, error)
}

type MissingItemHandler func(ctx context.Context, pointers []string) error

type peeker[T any] struct {
	client rueidis.Client

	max    int64
	opName string

	isMillisecondPrecision bool

	handleMissingItems MissingItemHandler
	maker              func() *T
	keyMetadataHash    string
}

type peekerOpt[T any] func(p *peeker[T])

func WithPeekerClient[T any](client rueidis.Client) peekerOpt[T] {
	return func(p *peeker[T]) {
		p.client = client
	}
}

func WithPeekerMaxPeekSize[T any](max int) peekerOpt[T] {
	return func(p *peeker[T]) {
		p.max = int64(max)
	}
}

func WithPeekerOpName[T any](opName string) peekerOpt[T] {
	return func(p *peeker[T]) {
		p.opName = opName
	}
}

func WithPeekerMillisecondPrecision[T any](isMillisecondPrecision bool) peekerOpt[T] {
	return func(p *peeker[T]) {
		p.isMillisecondPrecision = isMillisecondPrecision
	}
}

func WithPeekerMetadataHashKey[T any](keyMetadataHash string) peekerOpt[T] {
	return func(p *peeker[T]) {
		p.keyMetadataHash = keyMetadataHash
	}
}

func WithPeekerHandleMissingItems[T any](handler MissingItemHandler) peekerOpt[T] {
	return func(p *peeker[T]) {
		p.handleMissingItems = handler
	}
}

func WithPeekerMaker[T any](maker func() *T) peekerOpt[T] {
	return func(p *peeker[T]) {
		p.maker = maker
	}
}

func NewPeeker[T any](opt ...peekerOpt[T]) Peeker[T] {
	p := &peeker[T]{}
	for _, o := range opt {
		o(p)
	}

	return p
}
