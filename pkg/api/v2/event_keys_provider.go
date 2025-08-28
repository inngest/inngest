package apiv2

import (
	"context"
	"time"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventKeysProvider interface {
	GetEventKeys(ctx context.Context) ([]*apiv2.EventKey, error)
}

type eventKeysProvider []string

func NewEventKeysProvider(eventKeys []string) EventKeysProvider {
	return eventKeysProvider(eventKeys)
}

func (keys eventKeysProvider) GetEventKeys(ctx context.Context) ([]*apiv2.EventKey, error) {
	var eventKeys []*apiv2.EventKey
	
	for _, key := range keys {
		eventKeys = append(eventKeys, &apiv2.EventKey{
			Id:          "",
			Name:        "",
			Environment: "dev",
			Key:         key,
			CreatedAt:   timestamppb.New(time.Now()),
		})
	}
	
	return eventKeys, nil
}