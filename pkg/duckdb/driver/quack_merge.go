package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
)

// This file drives a MERGE INTO statement's source rows over quack's
// SEND_DATA_REQUEST mechanism instead of interpolating them as literal SQL
// VALUES — the same win quack_append.go's QuackAppender gets for plain
// INSERTs, applied to MERGE INTO's WHEN MATCHED/WHEN NOT MATCHED upsert
// semantics. See quack_senddata.go's package doc comment for the underlying
// wire mechanism (scan_data_from_quack_client + SEND_DATA_REQUEST) both
// files share.
//
// This isn't a MERGE-into-a-remote-quack-attached-catalog feature (DuckDB's
// own Catalog::PlanMergeInto has no override for the "quack" catalog type,
// so ATTACH 'quack:...' AS rpc; MERGE INTO rpc.t USING <local data> ...
// fails with "Database type "quack" does not support MERGE INTO or ON
// CONFLICT" — confirmed against this exact build). It doesn't need to be:
// this client talks directly to the server process that owns the DuckLake
// table, so the MERGE's target is that server's own ordinary local table,
// not a cross-process "remote" catalog attachment — scan_data_from_quack_client
// is just an ordinary table-valued source for it, the same as it already is
// for QuackAppender's INSERT.

// QuackMergeConfig describes the MERGE INTO statement a QuackMergeAppender
// drives against catalog.schema.table: MERGE INTO <table> AS t USING
// (SELECT * FROM scan_data_from_quack_client(...) [QUALIFY ROW_NUMBER()
// OVER (PARTITION BY <DedupKeys> ORDER BY <DedupOrderBy> DESC) = 1]) AS s ON
// <On> [WHEN MATCHED THEN UPDATE SET <UpdateSet>] WHEN NOT MATCHED THEN
// INSERT (<columns>) VALUES (s.<columns>...).
type QuackMergeConfig struct {
	// On is the ON condition's SQL text, referencing the target as "t" and
	// the source as "s" — e.g. "t.account_id = s.account_id AND t.run_id =
	// s.run_id". Required.
	On string
	// UpdateSet is the WHEN MATCHED THEN UPDATE SET clause's SQL text, e.g.
	// "status = s.status, updated_at = s.updated_at". Empty omits the WHEN
	// MATCHED clause entirely, so a matched row is left untouched (only
	// WHEN NOT MATCHED THEN INSERT runs).
	UpdateSet string
	// DedupKeys, if non-empty, lets one Flush's batch safely contain more
	// than one row for the same merge key: the USING subquery groups by
	// these column Names (matched against the columns passed to
	// NewQuackMergeAppender) and, per group, keeps only the row that sorts
	// last under DedupOrderBy — via QUALIFY ROW_NUMBER() OVER (PARTITION BY
	// <DedupKeys> ORDER BY <DedupOrderBy> DESC) = 1, evaluated before the ON
	// condition ever sees the source. Without this, a batch with two
	// same-key rows is unsafe: MERGE INTO evaluates WHEN NOT MATCHED against
	// the target's state as of the statement's start, not row-by-row, so two
	// source rows sharing a not-yet-existing key would both take the INSERT
	// branch and silently produce two target rows for one key. Empty (the
	// default) applies no dedup — the caller must guarantee key uniqueness
	// within a batch itself, as every caller predating this field already
	// does.
	DedupKeys []string
	// DedupOrderBy is the SQL expression DedupKeys' QUALIFY orders by,
	// descending, to choose which same-key row survives — referencing
	// source columns unqualified (e.g. "updated_at"), since it runs inside
	// the USING subquery before the "s" alias is assigned. Typically
	// whichever column best reflects "most recent state" for the caller's
	// rows. Required whenever DedupKeys is non-empty; ignored otherwise.
	DedupOrderBy string
}

// QuackMergeColumn names one column of a QuackMergeAppender's source rows.
// Unlike QuackAppender's columns (matched to the target table by position
// alone, since a plain INSERT ... SELECT * needs nothing else), a MERGE's
// On/UpdateSet SQL must be able to reference the source's columns by name
// ("s.<Name>"), so each one needs a real, caller-chosen name — it only has
// to be unique within one QuackMergeAppender, not match the target table's
// own column names.
type QuackMergeColumn struct {
	Name string
	Kind QuackColumnKind
}

// QuackMergeAppender bulk-merges rows into one table over quack's
// SEND_DATA_REQUEST mechanism (see quack_senddata.go). AppendRow buffers
// rows in memory; Flush runs the MERGE with everything buffered as one
// batch. Not safe for concurrent use (one goroutine at a time per
// *QuackMergeAppender — see NewQuackMergeAppender/NewQuackAppender's doc
// comments for why concurrent *appenders*, each on its own instance, are
// fine).
type QuackMergeAppender struct {
	session *quackSession
	schema  string
	table   string
	cfg     QuackMergeConfig
	columns []QuackMergeColumn
	rows    [][]any

	// pooledConn is non-nil only when this appender came from
	// NewQuackMergeAppender (not NewQuackMergeAppenderFromConn) — see
	// QuackAppender.pooledConn's doc comment, which this shares in full.
	pooledConn *sql.Conn
}

// NewQuackMergeAppender returns a QuackMergeAppender for catalog.schema.table
// — see NewQuackAppender's doc comment for db/catalog requirements and the
// pooled-connection lifetime hazard it fixes, both of which this shares in
// full (including the USE <catalog> caveat).
func NewQuackMergeAppender(ctx context.Context, db *sql.DB, catalog, schema, table string, cfg QuackMergeConfig, columns []QuackMergeColumn) (*QuackMergeAppender, error) {
	sqlConn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("duckdb: quack merge appender: acquiring connection: %w", err)
	}

	var appender *QuackMergeAppender
	err = sqlConn.Raw(func(driverConn any) error {
		var err error
		appender, err = newQuackMergeAppenderFromDriverConn(ctx, driverConn, catalog, schema, table, cfg, columns)
		return err
	})
	if err != nil {
		_ = sqlConn.Close()
		return nil, err
	}
	appender.pooledConn = sqlConn
	return appender, nil
}

// NewQuackMergeAppenderFromConn is NewQuackMergeAppender for a driver.Conn
// the caller already owns outright — see NewQuackAppenderFromConn's doc
// comment, which this shares in full.
func NewQuackMergeAppenderFromConn(ctx context.Context, driverConn driver.Conn, catalog, schema, table string, cfg QuackMergeConfig, columns []QuackMergeColumn) (*QuackMergeAppender, error) {
	return newQuackMergeAppenderFromDriverConn(ctx, driverConn, catalog, schema, table, cfg, columns)
}

func newQuackMergeAppenderFromDriverConn(ctx context.Context, driverConn any, catalog, schema, table string, cfg QuackMergeConfig, columns []QuackMergeColumn) (*QuackMergeAppender, error) {
	sess, err := resolveQuackSessionForAppender(ctx, driverConn, catalog, "quack merge appender")
	if err != nil {
		return nil, err
	}
	return &QuackMergeAppender{session: sess, schema: schema, table: table, cfg: cfg, columns: columns}, nil
}

// AppendRow buffers one row. len(vals) must equal len(columns) from
// NewQuackMergeAppender, in the same order. No I/O happens here — call
// Flush or Close to actually run the MERGE.
func (a *QuackMergeAppender) AppendRow(vals ...any) error {
	if len(vals) != len(a.columns) {
		return fmt.Errorf("duckdb: quack merge appender: got %d values, expected %d columns", len(vals), len(a.columns))
	}
	a.rows = append(a.rows, vals)
	return nil
}

// Flush runs the MERGE with every buffered row as one SEND_DATA_REQUEST
// batch and clears the buffer. A no-op if nothing is buffered.
func (a *QuackMergeAppender) Flush(ctx context.Context) error {
	if len(a.rows) == 0 {
		return nil
	}
	if err := a.session.sendDataMerge(ctx, a.schema, a.table, a.cfg, a.columns, a.rows); err != nil {
		return err
	}
	a.rows = a.rows[:0]
	return nil
}

// Close flushes any remaining buffered rows, then — for a
// QuackMergeAppender from NewQuackMergeAppender — releases the pooled
// connection it has held open since construction back to db's pool. The
// appender must not be used afterward.
func (a *QuackMergeAppender) Close(ctx context.Context) error {
	flushErr := a.Flush(ctx)
	if a.pooledConn != nil {
		if closeErr := a.pooledConn.Close(); closeErr != nil && flushErr == nil {
			return closeErr
		}
	}
	return flushErr
}

// buildQuackSendDataMergeSQL builds the MERGE INTO statement that opens a
// stream: its USING clause selects from scan_data_from_quack_client, whose
// second argument is a NULL cast to a STRUCT spelling out the batch's real
// column names and types — unlike buildQuackSendDataInsertSQL's positional
// c0/c1/..., cfg.On/cfg.UpdateSet need to reference these columns by name
// ("s.<name>"), so the STRUCT prototype carries the caller's chosen names
// instead of placeholders.
func buildQuackSendDataMergeSQL(schema, table string, cfg QuackMergeConfig, columns []QuackMergeColumn, streamID string) (string, error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("duckdb: quack merge appender: no columns")
	}
	if cfg.On == "" {
		return "", fmt.Errorf("duckdb: quack merge appender: On condition is required")
	}
	if len(cfg.DedupKeys) > 0 && cfg.DedupOrderBy == "" {
		return "", fmt.Errorf("duckdb: quack merge appender: DedupOrderBy is required when DedupKeys is set")
	}

	fields := make([]string, len(columns))
	names := make([]string, len(columns))
	insertValues := make([]string, len(columns))
	knownColumns := make(map[string]bool, len(columns))
	for i, c := range columns {
		typeName, err := quackSendDataColumnType(c.Kind)
		if err != nil {
			return "", err
		}
		ident := quackIdentifier(c.Name)
		fields[i] = fmt.Sprintf("%s %s", ident, typeName)
		names[i] = ident
		insertValues[i] = "s." + ident
		knownColumns[c.Name] = true
	}

	dedupKeys := make([]string, len(cfg.DedupKeys))
	for i, k := range cfg.DedupKeys {
		if !knownColumns[k] {
			return "", fmt.Errorf("duckdb: quack merge appender: DedupKeys references unknown column %q", k)
		}
		dedupKeys[i] = quackIdentifier(k)
	}

	tableRef := quackIdentifier(table)
	if schema != "" {
		tableRef = quackIdentifier(schema) + "." + tableRef
	}
	streamIDLiteral, err := encodeLiteral(streamID)
	if err != nil {
		return "", fmt.Errorf("duckdb: quack merge appender: encoding stream id: %w", err)
	}

	usingSelect := fmt.Sprintf("SELECT * FROM scan_data_from_quack_client(%s, NULL::STRUCT(%s), ordered := true)",
		streamIDLiteral, strings.Join(fields, ", "))
	if len(dedupKeys) > 0 {
		usingSelect = fmt.Sprintf("%s\nQUALIFY ROW_NUMBER() OVER (PARTITION BY %s ORDER BY %s DESC) = 1",
			usingSelect, strings.Join(dedupKeys, ", "), cfg.DedupOrderBy)
	}

	var matchedClause string
	if cfg.UpdateSet != "" {
		matchedClause = fmt.Sprintf("WHEN MATCHED THEN UPDATE SET %s\n", cfg.UpdateSet)
	}

	return fmt.Sprintf(
		"MERGE INTO %s AS t USING (%s) AS s ON %s\n%sWHEN NOT MATCHED THEN INSERT (%s) VALUES (%s);",
		tableRef, usingSelect, cfg.On, matchedClause,
		strings.Join(names, ", "), strings.Join(insertValues, ", "),
	), nil
}

// sendDataMerge runs a MERGE INTO schema.table with every row in rows as its
// USING source, streamed over quack's SEND_DATA_REQUEST mechanism (see
// quack_senddata.go's sendDataDrive, which this and sendDataAppend share).
func (s *quackSession) sendDataMerge(ctx context.Context, schema, table string, cfg QuackMergeConfig, columns []QuackMergeColumn, rows [][]any) error {
	streamID, err := generateQuackStreamID()
	if err != nil {
		return fmt.Errorf("duckdb: quack merge appender: generating stream id: %w", err)
	}
	sql, err := buildQuackSendDataMergeSQL(schema, table, cfg, columns, streamID)
	if err != nil {
		return err
	}
	kinds := make([]QuackColumnKind, len(columns))
	for i, c := range columns {
		kinds[i] = c.Kind
	}
	return s.sendDataDrive(ctx, sql, streamID, kinds, rows)
}
