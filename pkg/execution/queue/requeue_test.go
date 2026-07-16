package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestProducerRequeueByJobIDRejectsInvalidScope(t *testing.T) {
	ctx := context.Background()
	shard := &mockShardForIterator{name: "shard-a"}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	producer, err := New(ctx, "test", registry)
	require.NoError(t, err)

	tests := []struct {
		name    string
		scope   Scope
		wantErr string
	}{
		{
			name:    "missing account ID",
			scope:   Scope{EnvID: uuid.New(), FunctionID: uuid.New()},
			wantErr: "missing account ID",
		},
		{
			name:    "missing env ID",
			scope:   Scope{AccountID: uuid.New(), FunctionID: uuid.New()},
			wantErr: "missing env ID",
		},
		{
			name:    "missing function ID",
			scope:   Scope{AccountID: uuid.New(), EnvID: uuid.New()},
			wantErr: "missing function ID",
		},
		{
			name:    "missing all IDs",
			scope:   Scope{},
			wantErr: "missing account ID",
		},
	}

	for _, isSystem := range []bool{true, false} {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s/system=%t", tt.name, isSystem), func(t *testing.T) {
				tt.scope.IsSystem = isSystem
				err := producer.RequeueByJobID(ctx, tt.scope, shard.Name(), "job-id", time.Now())
				require.EqualError(t, err, tt.wantErr)
			})
		}
	}
}
