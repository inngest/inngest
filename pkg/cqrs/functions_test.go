package cqrs

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunction_InngestFunction(t *testing.T) {
	tests := []struct {
		name    string
		fn      Function
		wantErr bool
	}{
		{
			name: "valid function config",
			fn: Function{
				ID:        uuid.New(),
				EnvID:     uuid.New(),
				AppID:     uuid.New(),
				Slug:      "test-function",
				Name:      "Test Function",
				Config:    json.RawMessage(`{"triggers": [{"event": "test.event"}]}`),
				CreatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid json config",
			fn: Function{
				ID:        uuid.New(),
				EnvID:     uuid.New(),
				AppID:     uuid.New(),
				Slug:      "test-function",
				Name:      "Test Function",
				Config:    json.RawMessage(`{invalid json`),
				CreatedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "empty config",
			fn: Function{
				ID:        uuid.New(),
				EnvID:     uuid.New(),
				AppID:     uuid.New(),
				Slug:      "test-function",
				Name:      "Test Function",
				Config:    json.RawMessage(`{}`),
				CreatedAt: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn.InngestFunction()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestFunction_IsArchived(t *testing.T) {
	now := time.Now()
	weirdTime, err := time.Parse("0001-01-01T00:00:00Z", "0001-01-01T00:00:00Z")
	require.NoError(t, err)

	tests := []struct {
		name       string
		archivedAt time.Time
		want       bool
	}{
		{
			name:       "zero time archived_at",
			archivedAt: time.Time{},
			want:       false,
		},
		{
			name:       "beginning of time archived_at",
			archivedAt: weirdTime,
			want:       false,
		},
		{
			name:       "past time archived_at",
			archivedAt: now.Add(-time.Hour),
			want:       true,
		},
		{
			name:       "current time archived_at",
			archivedAt: now,
			want:       true,
		},
		{
			name:       "future time archived_at",
			archivedAt: now.Add(time.Hour),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Function{
				ID:         uuid.New(),
				EnvID:      uuid.New(),
				AppID:      uuid.New(),
				Slug:       "test-function",
				Name:       "Test Function",
				Config:     json.RawMessage(`{}`),
				CreatedAt:  time.Now(),
				ArchivedAt: tt.archivedAt,
			}

			got := f.IsArchived()
			assert.Equal(t, tt.want, got)
		})
	}
}
