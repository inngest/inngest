package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/run"
	"github.com/oklog/ulid/v2"
)

const eventColumns = "account_id, env_id, internal_id, received_at, source, source_id, event_id, event_name, event_data, event_v, event_ts"

// GetEventsByExpressions returns every event matching all of the given CEL
// expressions (ANDed together). Like the interface itself, it takes no
// accountID/workspaceID -- callers must have their CEL expressions do that
// scoping.
//
// The SQL built from cel is only a superset prefilter (see celmatch.go), so
// like pkg/cqrs/manager every candidate is re-checked with
// run.ExpressionHandler.MatchEventExpressions, which also means (like
// there) an expression list with no event.* expression at all matches
// nothing.
func (m *Manager) GetEventsByExpressions(ctx context.Context, cel []string) ([]*cqrs.Event, error) {
	expHandler, err := run.NewExpressionHandler(ctx, run.WithExpressionHandlerExpressions(cel))
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: parsing CEL filter: %w", err)
	}
	if !expHandler.HasEventFilters() {
		return []*cqrs.Event{}, nil
	}

	filters, err := CELEventTableFilters(ctx, cel)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: converting CEL filter: %w", err)
	}

	where, args, err := RenderWhereSQL(filters)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: rendering CEL filter: %w", err)
	}

	query := fmt.Sprintf("SELECT %s FROM %s.events", eventColumns, driver.DuckLakeAlias)
	if where != "" {
		query += " WHERE " + where
	}
	query += " ORDER BY received_at DESC;"

	candidates, err := m.queryEvents(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	out := []*cqrs.Event{}
	for _, evt := range candidates {
		ok, err := expHandler.MatchEventExpressions(ctx, evt.GetEvent())
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, evt)
		}
	}
	return out, nil
}

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
		eventColumns, driver.DuckLakeAlias, strings.Join(placeholders, ", "),
	)
	return m.queryEvents(ctx, query, args...)
}

func (m *Manager) GetEvents(ctx context.Context, accountID, workspaceID uuid.UUID, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		`SELECT %s FROM %s.events WHERE account_id = ? AND env_id = ? AND received_at <= ? AND received_at >= ?`,
		eventColumns, driver.DuckLakeAlias,
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
	cel, err := eventsCELWhere(ctx, opts.CEL)
	if err != nil {
		return nil, err
	}
	query += " AND (" + cel.where + ")"
	args = append(args, cel.args...)

	if cel.matcher == nil {
		return m.queryEvents(ctx, query+" ORDER BY internal_id DESC LIMIT ?;", append(args, opts.Limit)...)
	}

	// The CEL SQL is only a prefilter, so page through candidates in
	// internal_id order (the same keyset the request cursor uses) until
	// opts.Limit of them pass the Go matcher or they run out.
	batch := int(celCandidateBatch(uint(opts.Limit)))
	out := []*cqrs.Event{}
	var after *ulid.ULID
	for {
		batchQuery, batchArgs := query, append([]any{}, args...)
		if after != nil {
			batchQuery += " AND internal_id < ?"
			batchArgs = append(batchArgs, after.String())
		}
		candidates, err := m.queryEvents(ctx, batchQuery+" ORDER BY internal_id DESC LIMIT ?;", append(batchArgs, batch)...)
		if err != nil {
			return nil, err
		}
		for _, evt := range candidates {
			ok, err := cel.matcher.MatchEventExpressions(ctx, evt.GetEvent())
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			out = append(out, evt)
			if len(out) >= opts.Limit {
				return out, nil
			}
		}
		if len(candidates) < batch {
			return out, nil
		}
		after = &candidates[len(candidates)-1].ID
	}
}

// GetEventsCount backs EventsConnection.totalCount. Unlike GetEvents, the
// cursor is never applied here — total count must not shrink as pagination
// advances.
func (m *Manager) GetEventsCount(ctx context.Context, accountID, workspaceID uuid.UUID, opts cqrs.WorkspaceEventsOpts) (int64, error) {
	if err := opts.Validate(); err != nil {
		return 0, err
	}

	query := fmt.Sprintf(
		`FROM %s.events WHERE account_id = ? AND env_id = ? AND received_at <= ? AND received_at >= ?`,
		driver.DuckLakeAlias,
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
	cel, err := eventsCELWhere(ctx, opts.CEL)
	if err != nil {
		return 0, err
	}
	query += " AND (" + cel.where + ")"
	args = append(args, cel.args...)

	if cel.matcher == nil {
		return scanCount(m.db.QueryRowContext(ctx, "SELECT COUNT(*) AS c "+query+";", args...))
	}

	// The CEL SQL is only a prefilter, so every candidate has to be
	// re-checked in Go rather than counted in SQL.
	candidates, err := m.queryEvents(ctx, "SELECT "+eventColumns+" "+query+";", args...)
	if err != nil {
		return 0, err
	}
	var n int64
	for _, evt := range candidates {
		ok, err := cel.matcher.MatchEventExpressions(ctx, evt.GetEvent())
		if err != nil {
			return 0, err
		}
		if ok {
			n++
		}
	}
	return n, nil
}

// eventsCEL is a WorkspaceEventsOpts.CEL filter: a superset WHERE-clause
// prefilter against the events table's own columns, plus the handler every
// candidate it selects still has to pass MatchEventExpressions against
// (nil when the CEL has no event.* expression to match).
type eventsCEL struct {
	where   string
	args    []any
	matcher *run.ExpressionHandler
}

// eventsCELWhere converts a single (possibly multi-line) CEL expression
// into an eventsCEL. An empty cel selects everything.
func eventsCELWhere(ctx context.Context, cel string) (eventsCEL, error) {
	expHandler, err := run.NewExpressionHandler(ctx, run.WithExpressionHandlerBlob(cel, "\n"))
	if err != nil {
		return eventsCEL{}, fmt.Errorf("duckdbquery: parsing CEL filter: %w", err)
	}

	res := eventsCEL{where: "TRUE"}
	if expHandler.HasEventFilters() {
		res.matcher = expHandler
	}

	filters, err := CELEventTableFilters(ctx, expHandler.EventExprList)
	if err != nil {
		return eventsCEL{}, fmt.Errorf("duckdbquery: converting CEL filter: %w", err)
	}
	where, args, err := RenderWhereSQL(filters)
	if err != nil {
		return eventsCEL{}, fmt.Errorf("duckdbquery: rendering CEL filter: %w", err)
	}
	if where != "" {
		res.where, res.args = where, args
	}
	return res, nil
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
