package apiv2

import (
	"context"
	"time"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SigningKeysProvider interface {
	GetSigningKeys(ctx context.Context) ([]*apiv2.SigningKey, error)
}

type signingKeysProvider string

func NewSigningKeysProvider(signingKey string) SigningKeysProvider {
	return signingKeysProvider(signingKey)
}

func (key signingKeysProvider) GetSigningKeys(ctx context.Context) ([]*apiv2.SigningKey, error) {
	return []*apiv2.SigningKey{{
		Id:          "",
		Name:        "",
		Environment: "dev",
		Key:         string(key),
		CreatedAt:   timestamppb.New(time.Now()),
	}}, nil
}
