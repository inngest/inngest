package duckdbquery

import (
	"context"
	"database/sql"
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

// GetEventsCount backs EventsConnection.totalCount (events_connection.go) —
// previously unoverridden, which silently fell through to the primary
// (SQLite/Postgres) manager while GetEvents' edges came from DuckDB, so the
// count and the list could disagree. Mirrors pkg/cqrs/manager's own
// GetEventsCount: cursor is never applied to the count (total count must not
// shrink as pagination advances), unlike GetEvents.
func (m *Manager) GetEventsCount(ctx context.Context, accountID, workspaceID uuid.UUID, opts cqrs.WorkspaceEventsOpts) (int64, error) {
	if err := opts.Validate(); err != nil {
		return 0, err
	}

	query := fmt.Sprintf(
		`SELECT COUNT(*) AS c FROM %s.events WHERE account_id = ? AND env_id = ? AND received_at <= ? AND received_at >= ?`,
		duckdb.DuckLakeAlias,
	)
	args := []any{accountID.String(), workspaceID.String(), opts.Newest, opts.Oldest}

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
	query += ";"

	return scanCount(m.db.QueryRowContext(ctx, query, args...))
}

func (m *Manager) queryEvents(ctx context.Context, query string, args ...any) ([]*cqrs.Event, error) {
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying events: %w", err)
	}
	defer rows.Close()

	var out []*cqrs.Event
	for rows.Next() {
		evt, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("duckdbquery: scanning event row: %w", err)
		}
		out = append(out, evt)
	}
	return out, rows.Err()
}

// scanEvent scans one row of an eventColumns-shaped SELECT into a
// *cqrs.Event. Destination order must match eventColumns exactly.
func scanEvent(rows *sql.Rows) (*cqrs.Event, error) {
	var (
		rawAccountID, rawWorkspaceID, rawInternalID, rawReceivedAt any
		source                                                     string
		rawSourceID                                                any
		eventID, eventName                                         string
		rawEventData                                               any
		eventV                                                     string
		rawEventTS                                                 any
	)
	if err := rows.Scan(
		&rawAccountID, &rawWorkspaceID, &rawInternalID, &rawReceivedAt,
		&source, &rawSourceID, &eventID, &eventName, &rawEventData, &eventV, &rawEventTS,
	); err != nil {
		return nil, err
	}

	accountID, err := uuidColumn(rawAccountID, "account_id")
	if err != nil {
		return nil, err
	}
	workspaceID, err := uuidColumn(rawWorkspaceID, "env_id")
	if err != nil {
		return nil, err
	}
	internalID, err := ulidColumn(rawInternalID, "internal_id")
	if err != nil {
		return nil, err
	}
	receivedAt, err := asTimestamp(rawReceivedAt, "received_at")
	if err != nil {
		return nil, err
	}
	sourceID, err := nullableUUIDColumn(rawSourceID, "source_id")
	if err != nil {
		return nil, err
	}
	data, err := asMap(rawEventData, "event_data")
	if err != nil {
		return nil, err
	}
	eventTS, err := asTimestamp(rawEventTS, "event_ts")
	if err != nil {
		return nil, err
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
