package devserver

import (
	"context"
	"database/sql"
	"errors"

	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

type eventProvider struct{ reader cqrs.EventReader }

func NewEventProvider(reader cqrs.EventReader) apiv2.EventProvider {
	return &eventProvider{reader: reader}
}

func (p *eventProvider) GetEvent(ctx context.Context, id ulid.ULID) (*cqrs.Event, error) {
	evt, err := p.reader.GetEventByInternalID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apiv2.ErrEventNotFound
	}
	return evt, err
}

func (p *eventProvider) GetEvents(ctx context.Context, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
	return p.reader.GetEvents(ctx, consts.DevServerAccountID, consts.DevServerEnvID, opts)
}
