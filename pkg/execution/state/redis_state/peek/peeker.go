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

	Cursor int64
}

// Peeker defines the interface for peeking operations on queues
type Peeker[T any] interface {
	Peek(ctx context.Context, keyOrderedPointerSet string, opt ...Option) (*Result[T], error)
	PeekPointer(ctx context.Context, keyOrderedPointerSet string, opt ...Option) ([]string, error)
	PeekUUIDPointer(ctx context.Context, keyOrderedPointerSet string, opt ...Option) ([]uuid.UUID, error)
}

type MissingItemHandler func(ctx context.Context, pointers []string) error

// peekerOption represents a non-generic peeker configuration option
type peekerOption func(pb *peekerBase)

// peekerBase contains the non-generic fields shared by all peeker instances
type peekerBase struct {
	client rueidis.Client

	max    int64
	opName string

	isMillisecondPrecision bool

	handleMissingItems MissingItemHandler
	keyMetadataHash    string
}

type peeker[T any] struct {
	peekerBase
	maker func() *T
}

func WithPeekerClient(client rueidis.Client) peekerOption {
	return func(pb *peekerBase) {
		pb.client = client
	}
}

func WithPeekerMaxPeekSize(max int) peekerOption {
	return func(pb *peekerBase) {
		pb.max = int64(max)
	}
}

func WithPeekerOpName(opName string) peekerOption {
	return func(pb *peekerBase) {
		pb.opName = opName
	}
}

func WithPeekerMillisecondPrecision(isMillisecondPrecision bool) peekerOption {
	return func(pb *peekerBase) {
		pb.isMillisecondPrecision = isMillisecondPrecision
	}
}

func WithPeekerMetadataHashKey(keyMetadataHash string) peekerOption {
	return func(pb *peekerBase) {
		pb.keyMetadataHash = keyMetadataHash
	}
}

func WithPeekerHandleMissingItems(handler MissingItemHandler) peekerOption {
	return func(pb *peekerBase) {
		pb.handleMissingItems = handler
	}
}

// NewPeeker creates a new peeker with the given maker function and options.
// The maker function is required and specifies the type T, while options are type-erased.
func NewPeeker[T any](maker func() *T, opts ...peekerOption) Peeker[T] {
	p := &peeker[T]{
		maker: maker,
	}

	// Apply all non-generic options
	for _, opt := range opts {
		opt(&p.peekerBase)
	}

	return p
}
