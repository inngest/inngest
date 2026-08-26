package duckdbquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/oklog/ulid/v2"
)

const eventColumns = "account_id, env_id, internal_id, received_at, source, source_id, event_id, event_name, event_data, event_v, event_ts"

func (m *Manager) GetEventsByInternalIDs(ctx context.Context, ids []ulid.ULID) ([]*cqrs.Event, error) {
	if len(ids) == 0 {
		return []*cqrs.Event{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id.String()
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s.events WHERE internal_id IN (%s);",
		eventColumns, duckdb.DuckLakeAlias, strings.Join(placeholders, ", "),
	)
	return m.queryEvents(ctx, query, args...)
}

func (m *Manager) GetEvents(ctx context.Context, accountID, workspaceID uuid.UUID, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		`SELECT %s FROM %s.events WHERE account_id = ? AND env_id = ? AND received_at <= ? AND received_at >= ?`,
		eventColumns, duckdb.DuckLakeAlias,
	)
	args := []any{accountID.String(), workspaceID.String(), opts.Newest, opts.Oldest}

	if opts.Cursor != nil {
		// internal_id is a ULID string; lexicographic comparison matches
		// chronological order for same-length ULIDs.
		query += " AND internal_id < ?"
		args = append(args, opts.Cursor.String())
	}
	if len(opts.Names) > 0 {
		placeholders := make([]string, len(opts.Names))
		for i, name := range opts.Names {
			placeholders[i] = "?"
			args = append(args, name)
		}
		query += fmt.Sprintf(" AND event_name IN (%s)", strings.Join(placeholders, ", "))
	}
	if !opts.IncludeInternalEvents {
		query += " AND event_name NOT LIKE 'inngest/%'"
	}
	query += " ORDER BY internal_id DESC LIMIT ?;"
	args = append(args, opts.Limit)

	return m.queryEvents(ctx, query, args...)
}

func (m *Manager) queryEvents(ctx context.Context, query string, args ...any) ([]*cqrs.Event, error) {
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying events: %w", err)
	}
	defer rows.Close()

	named, err := scanNamedRows(rows)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: scanning event rows: %w", err)
	}

	out := make([]*cqrs.Event, 0, len(named))
	for _, row := range named {
		evt, err := eventFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, evt)
	}
	return out, nil
}

func eventFromRow(row map[string]any) (*cqrs.Event, error) {
	accountID, err := uuidField(row, "account_id")
	if err != nil {
		return nil, err
	}
	workspaceID, err := uuidField(row, "env_id")
	if err != nil {
		return nil, err
	}
	internalID, err := ulidField(row, "internal_id")
	if err != nil {
		return nil, err
	}
	receivedAt, err := timeField(row, "received_at")
	if err != nil {
		return nil, err
	}
	source, err := stringField(row, "source")
	if err != nil {
		return nil, err
	}
	sourceID, err := nullableUUIDField(row, "source_id")
	if err != nil {
		return nil, err
	}
	eventID, err := stringField(row, "event_id")
	if err != nil {
		return nil, err
	}
	eventName, err := stringField(row, "event_name")
	if err != nil {
		return nil, err
	}
	eventV, err := stringField(row, "event_v")
	if err != nil {
		return nil, err
	}
	eventTS, err := timeField(row, "event_ts")
	if err != nil {
		return nil, err
	}

	data, ok := row["event_data"].(map[string]any)
	if !ok {
		data = map[string]any{}
	}

	return &cqrs.Event{
		ID:           internalID,
		AccountID:    accountID,
		WorkspaceID:  workspaceID,
		Source:       source,
		SourceID:     sourceID,
		ReceivedAt:   receivedAt,
		EventID:      eventID,
		EventName:    eventName,
		EventData:    data,
		EventTS:      eventTS.UnixMilli(),
		EventVersion: eventV,
	}, nil
}
