package inmemory

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngestctl/pkg/execution/state"
	"github.com/oklog/ulid"
	"github.com/stretchr/testify/assert"
)

func newULID() ulid.ULID {
	return ulid.MustNew(ulid.Now(), rand.Reader)
}

func TestInMemorySaveOutput(t *testing.T) {
	ctx := context.Background()

	sm := NewStateManager()

	i := state.Identifier{
		WorkflowID: uuid.New(),
		RunID:      newULID(),
	}

	s, err := sm.Load(ctx, i)
	assert.Nil(t, err)
	assert.Equal(t, i.WorkflowID, s.(*memstate).workflowID)
	assert.Equal(t, i.RunID, s.(*memstate).runID)

	data := map[string]interface{}{"ok": true}
	_, err = sm.SaveActionOutput(ctx, i, "1", data)
	assert.Nil(t, err)

	s, err = sm.Load(ctx, i)
	assert.Nil(t, err)
	assert.Equal(t, i.WorkflowID, s.(*memstate).workflowID)
	assert.Equal(t, i.RunID, s.(*memstate).runID)
	assert.Equal(t, 1, len(s.(*memstate).actions))
	assert.Equal(t, data, s.(*memstate).actions["1"])
}
