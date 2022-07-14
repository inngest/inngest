package ocsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"sync"

	"go.opencensus.io/trace"
)

type conn interface {
	driver.Pinger
	driver.Execer
	driver.ExecerContext
	driver.Queryer
	driver.QueryerContext
	driver.Conn
	driver.ConnPrepareContext
	driver.ConnBeginTx
}

var (
	regMu              sync.Mutex
	attrMissingContext = trace.StringAttribute("ocsql.warning", "missing upstream context")
	attrDeprecated     = trace.StringAttribute("ocsql.warning", "database driver uses deprecated features")

	// Compile time assertions
	_ driver.Driver                         = &ocDriver{}
	_ conn                                  = &ocConn{}
	_ driver.NamedValueChecker              = &ocConn{}
	_ driver.Result                         = &ocResult{}
	_ driver.Stmt                           = &ocStmt{}
	_ driver.StmtExecContext                = &ocStmt{}
	_ driver.StmtQueryContext               = &ocStmt{}
	_ driver.Rows                           = &ocRows{}
	_ driver.RowsNextResultSet              = &ocRows{}
	_ driver.RowsColumnTypeDatabaseTypeName = &ocRows{}
	_ driver.RowsColumnTypeLength           = &ocRows{}
	_ driver.RowsColumnTypeNullable         = &ocRows{}
	_ driver.RowsColumnTypePrecisionScale   = &ocRows{}
)

// Register initializes and registers our ocsql wrapped database driver
// identified by its driverName and using provided TraceOptions. On success it
// returns the generated driverName to use when calling sql.Open.
// It is possible to register multiple wrappers for the same database driver if
// needing different TraceOptions for different connections.
func Register(driverName string, options ...TraceOption) (string, error) {
	return RegisterWithSource(driverName, "", options...)
}

// RegisterWithSource initializes and registers our ocsql wrapped database driver
// identified by its driverName, using provided TraceOptions.
// source is useful if some drivers do not accept the empty string when opening the DB.
// On success it returns the generated driverName to use when calling sql.Open.
// It is possible to register multiple wrappers for the same database driver if
// needing different TraceOptions for different connections.
func RegisterWithSource(driverName string, source string, options ...TraceOption) (string, error) {
	// retrieve the driver implementation we need to wrap with instrumentation
	db, err := sql.Open(driverName, source)
	if err != nil {
		return "", err
	}
	dri := db.Driver()
	if err = db.Close(); err != nil {
		return "", err
	}

	regMu.Lock()
	defer regMu.Unlock()

	// Since we might want to register multiple ocsql drivers to have different
	// TraceOptions, but potentially the same underlying database driver, we
	// cycle through to find available driver names.
	driverName = driverName + "-ocsql-"
	for i := int64(0); i < 100; i++ {
		var (
			found   = false
			regName = driverName + strconv.FormatInt(i, 10)
		)
		for _, name := range sql.Drivers() {
			if name == regName {
				found = true
			}
		}
		if !found {
			sql.Register(regName, Wrap(dri, options...))
			return regName, nil
		}
	}
	return "", errors.New("unable to register driver, all slots have been taken")
}

// Wrap takes a SQL driver and wraps it with OpenCensus instrumentation.
func Wrap(d driver.Driver, options ...TraceOption) driver.Driver {
	o := TraceOptions{}
	for _, option := range options {
		option(&o)
	}
	if o.InstanceName == "" {
		o.InstanceName = defaultInstanceName
	} else {
		o.DefaultAttributes = append(o.DefaultAttributes, trace.StringAttribute("sql.instance", o.InstanceName))
	}
	if o.QueryParams && !o.Query {
		o.QueryParams = false
	}
	return wrapDriver(d, o)
}

// Open implements driver.Driver
func (d ocDriver) Open(name string) (driver.Conn, error) {
	c, err := d.parent.Open(name)
	if err != nil {
		return nil, err
	}
	return wrapConn(c, d.options), nil
}

// WrapConn allows an existing driver.Conn to be wrapped by ocsql.
func WrapConn(c driver.Conn, options ...TraceOption) driver.Conn {
	o := TraceOptions{}
	for _, option := range options {
		option(&o)
	}
	if o.InstanceName == "" {
		o.InstanceName = defaultInstanceName
	} else {
		o.DefaultAttributes = append(o.DefaultAttributes, trace.StringAttribute("sql.instance", o.InstanceName))
	}
	return wrapConn(c, o)
}

// ocConn implements driver.Conn
type ocConn struct {
	parent  driver.Conn
	options TraceOptions
}

func (c ocConn) Ping(ctx context.Context) (err error) {
	onDeferWithErr := recordCallStats(ctx, "go.sql.ping", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	if c.options.Ping && (c.options.AllowRoot || trace.FromContext(ctx) != nil) {
		var span *trace.Span
		ctx, span = trace.StartSpan(ctx, "sql:ping",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(c.options.Sampler),
		)
		if len(c.options.DefaultAttributes) > 0 {
			span.AddAttributes(c.options.DefaultAttributes...)
		}
		defer func() {
			if err != nil {
				span.SetStatus(trace.Status{
					Code:    trace.StatusCodeUnavailable,
					Message: err.Error(),
				})
			} else {
				span.SetStatus(trace.Status{Code: trace.StatusCodeOK})
			}
			span.End()
		}()
	}

	if pinger, ok := c.parent.(driver.Pinger); ok {
		err = pinger.Ping(ctx)
	}
	return
}

func (c ocConn) Exec(query string, args []driver.Value) (res driver.Result, err error) {
	onDeferWithErr := recordCallStats(context.Background(), "go.sql.exec", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	if exec, ok := c.parent.(driver.Execer); ok {
		if !c.options.AllowRoot {
			return exec.Exec(query, args)
		}

		ctx, span := trace.StartSpan(context.Background(), "sql:exec",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(c.options.Sampler),
		)
		attrs := make([]trace.Attribute, 0, len(c.options.DefaultAttributes)+2)
		attrs = append(attrs, c.options.DefaultAttributes...)
		attrs = append(
			attrs,
			attrDeprecated,
			trace.StringAttribute(
				"ocsql.deprecated", "driver does not support ExecerContext",
			),
		)
		if c.options.Query {
			attrs = append(attrs, trace.StringAttribute("sql.query", query))
			if c.options.QueryParams {
				attrs = append(attrs, paramsAttr(args)...)
			}
		}
		span.AddAttributes(attrs...)

		defer func() {
			setSpanStatus(span, c.options, err)
			span.End()
		}()

		if res, err = exec.Exec(query, args); err != nil {
			return nil, err
		}

		return ocResult{parent: res, ctx: ctx, options: c.options}, nil
	}

	return nil, driver.ErrSkip
}

func (c ocConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (res driver.Result, err error) {
	onDeferWithErr := recordCallStats(ctx, "go.sql.exec", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	if execCtx, ok := c.parent.(driver.ExecerContext); ok {
		parentSpan := trace.FromContext(ctx)
		if !c.options.AllowRoot && parentSpan == nil {
			return execCtx.ExecContext(ctx, query, args)
		}

		var span *trace.Span
		if parentSpan == nil {
			ctx, span = trace.StartSpan(ctx, "sql:exec",
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithSampler(c.options.Sampler),
			)
		} else {
			_, span = trace.StartSpan(ctx, "sql:exec",
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithSampler(c.options.Sampler),
			)
		}
		attrs := append([]trace.Attribute(nil), c.options.DefaultAttributes...)
		if c.options.Query {
			attrs = append(attrs, trace.StringAttribute("sql.query", query))
			if c.options.QueryParams {
				attrs = append(attrs, namedParamsAttr(args)...)
			}
		}
		span.AddAttributes(attrs...)

		defer func() {
			setSpanStatus(span, c.options, err)
			span.End()
		}()

		if res, err = execCtx.ExecContext(ctx, query, args); err != nil {
			return nil, err
		}

		return ocResult{parent: res, ctx: ctx, options: c.options}, nil
	}

	return nil, driver.ErrSkip
}

func (c ocConn) Query(query string, args []driver.Value) (rows driver.Rows, err error) {
	onDeferWithErr := recordCallStats(context.Background(), "go.sql.query", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	if queryer, ok := c.parent.(driver.Queryer); ok {
		if !c.options.AllowRoot {
			return queryer.Query(query, args)
		}

		ctx, span := trace.StartSpan(context.Background(), "sql:query",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(c.options.Sampler),
		)
		attrs := make([]trace.Attribute, 0, len(c.options.DefaultAttributes)+2)
		attrs = append(attrs, c.options.DefaultAttributes...)
		attrs = append(
			attrs,
			attrDeprecated,
			trace.StringAttribute(
				"ocsql.deprecated", "driver does not support QueryerContext",
			),
		)
		if c.options.Query {
			attrs = append(attrs, trace.StringAttribute("sql.query", query))
			if c.options.QueryParams {
				attrs = append(attrs, paramsAttr(args)...)
			}
		}
		span.AddAttributes(attrs...)

		defer func() {
			setSpanStatus(span, c.options, err)
			span.End()
		}()

		rows, err = queryer.Query(query, args)
		if err != nil {
			return nil, err
		}

		return wrapRows(ctx, rows, c.options), nil
	}

	return nil, driver.ErrSkip
}

func (c ocConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	onDeferWithErr := recordCallStats(ctx, "go.sql.query", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	if queryerCtx, ok := c.parent.(driver.QueryerContext); ok {
		parentSpan := trace.FromContext(ctx)
		if !c.options.AllowRoot && parentSpan == nil {
			return queryerCtx.QueryContext(ctx, query, args)
		}

		var span *trace.Span
		if parentSpan == nil {
			ctx, span = trace.StartSpan(ctx, "sql:query",
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithSampler(c.options.Sampler),
			)
		} else {
			_, span = trace.StartSpan(ctx, "sql:query",
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithSampler(c.options.Sampler),
			)
		}
		attrs := append([]trace.Attribute(nil), c.options.DefaultAttributes...)
		if c.options.Query {
			attrs = append(attrs, trace.StringAttribute("sql.query", query))
			if c.options.QueryParams {
				attrs = append(attrs, namedParamsAttr(args)...)
			}
		}
		span.AddAttributes(attrs...)

		defer func() {
			setSpanStatus(span, c.options, err)
			span.End()
		}()

		rows, err = queryerCtx.QueryContext(ctx, query, args)
		if err != nil {
			return nil, err
		}

		return wrapRows(ctx, rows, c.options), nil
	}

	return nil, driver.ErrSkip
}

func (c ocConn) Prepare(query string) (stmt driver.Stmt, err error) {
	onDeferWithErr := recordCallStats(context.Background(), "go.sql.prepare", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	if c.options.AllowRoot {
		_, span := trace.StartSpan(context.Background(), "sql:prepare",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(c.options.Sampler),
		)
		attrs := make([]trace.Attribute, 0, len(c.options.DefaultAttributes)+1)
		attrs = append(attrs, c.options.DefaultAttributes...)
		attrs = append(attrs, attrMissingContext)
		if c.options.Query {
			attrs = append(attrs, trace.StringAttribute("sql.query", query))
		}
		span.AddAttributes(attrs...)

		defer func() {
			setSpanStatus(span, c.options, err)
			span.End()
		}()
	}

	stmt, err = c.parent.Prepare(query)
	if err != nil {
		return nil, err
	}

	stmt = wrapStmt(stmt, query, c.options)
	return
}

func (c *ocConn) Close() error {
	return c.parent.Close()
}

func (c *ocConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.TODO(), driver.TxOptions{})
}

func (c *ocConn) PrepareContext(ctx context.Context, query string) (stmt driver.Stmt, err error) {
	onDeferWithErr := recordCallStats(ctx, "go.sql.prepare", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	var span *trace.Span
	attrs := append([]trace.Attribute(nil), c.options.DefaultAttributes...)
	if c.options.AllowRoot || trace.FromContext(ctx) != nil {
		ctx, span = trace.StartSpan(ctx, "sql:prepare",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(c.options.Sampler),
		)
		if c.options.Query {
			attrs = append(attrs, trace.StringAttribute("sql.query", query))
		}
		defer func() {
			setSpanStatus(span, c.options, err)
			span.End()
		}()
	}

	if prepCtx, ok := c.parent.(driver.ConnPrepareContext); ok {
		stmt, err = prepCtx.PrepareContext(ctx, query)
	} else {
		if span != nil {
			attrs = append(attrs, attrMissingContext)
		}
		stmt, err = c.parent.Prepare(query)
	}
	span.AddAttributes(attrs...)
	if err != nil {
		return nil, err
	}

	stmt = wrapStmt(stmt, query, c.options)
	return
}

func (c *ocConn) BeginTx(ctx context.Context, opts driver.TxOptions) (tx driver.Tx, err error) {
	onDeferWithErr := recordCallStats(ctx, "go.sql.begin", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	if !c.options.AllowRoot && trace.FromContext(ctx) == nil {
		if connBeginTx, ok := c.parent.(driver.ConnBeginTx); ok {
			return connBeginTx.BeginTx(ctx, opts)
		}
		return c.parent.Begin()
	}

	var span *trace.Span
	attrs := append([]trace.Attribute(nil), c.options.DefaultAttributes...)

	if ctx == nil || ctx == context.TODO() {
		ctx = context.Background()
		_, span = trace.StartSpan(ctx, "sql:begin_transaction",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(c.options.Sampler),
		)
		attrs = append(attrs, attrMissingContext)
	} else {
		_, span = trace.StartSpan(ctx, "sql:begin_transaction",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(c.options.Sampler),
		)
	}
	defer func() {
		if len(attrs) > 0 {
			span.AddAttributes(attrs...)
		}
		span.End()
	}()

	if connBeginTx, ok := c.parent.(driver.ConnBeginTx); ok {
		tx, err = connBeginTx.BeginTx(ctx, opts)
		setSpanStatus(span, c.options, err)
		if err != nil {
			return nil, err
		}
		return ocTx{parent: tx, ctx: ctx, options: c.options}, nil
	}

	attrs = append(
		attrs,
		attrDeprecated,
		trace.StringAttribute(
			"ocsql.deprecated", "driver does not support ConnBeginTx",
		),
	)
	tx, err = c.parent.Begin()
	setSpanStatus(span, c.options, err)
	if err != nil {
		return nil, err
	}
	return ocTx{parent: tx, ctx: ctx, options: c.options}, nil
}

func (c *ocConn) CheckNamedValue(nv *driver.NamedValue) (err error) {
	nvc, ok := c.parent.(driver.NamedValueChecker)
	if ok {
		return nvc.CheckNamedValue(nv)
	}
	nv.Value, err = driver.DefaultParameterConverter.ConvertValue(nv.Value)
	return err
}

// ocResult implements driver.Result
type ocResult struct {
	parent  driver.Result
	ctx     context.Context
	options TraceOptions
}

func (r ocResult) LastInsertId() (id int64, err error) {
	if r.options.LastInsertID {
		_, span := trace.StartSpan(r.ctx, "sql:last_insert_id",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(r.options.Sampler),
		)
		if len(r.options.DefaultAttributes) > 0 {
			span.AddAttributes(r.options.DefaultAttributes...)
		}
		defer func() {
			setSpanStatus(span, r.options, err)
			span.End()
		}()
	}

	id, err = r.parent.LastInsertId()
	return
}

func (r ocResult) RowsAffected() (cnt int64, err error) {
	if r.options.RowsAffected {
		_, span := trace.StartSpan(r.ctx, "sql:rows_affected",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(r.options.Sampler),
		)
		if len(r.options.DefaultAttributes) > 0 {
			span.AddAttributes(r.options.DefaultAttributes...)
		}
		defer func() {
			setSpanStatus(span, r.options, err)
			span.End()
		}()
	}

	cnt, err = r.parent.RowsAffected()
	return
}

// ocStmt implements driver.Stmt
type ocStmt struct {
	parent  driver.Stmt
	query   string
	options TraceOptions
}

func (s ocStmt) Exec(args []driver.Value) (res driver.Result, err error) {
	onDeferWithErr := recordCallStats(context.Background(), "go.sql.stmt.exec", s.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	if !s.options.AllowRoot {
		return s.parent.Exec(args)
	}

	ctx, span := trace.StartSpan(context.Background(), "sql:exec",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithSampler(s.options.Sampler),
	)
	attrs := make([]trace.Attribute, 0, len(s.options.DefaultAttributes)+2)
	attrs = append(attrs, s.options.DefaultAttributes...)
	attrs = append(
		attrs,
		attrDeprecated,
		trace.StringAttribute(
			"ocsql.deprecated", "driver does not support StmtExecContext",
		),
	)
	if s.options.Query {
		attrs = append(attrs, trace.StringAttribute("sql.query", s.query))
		if s.options.QueryParams {
			attrs = append(attrs, paramsAttr(args)...)
		}
	}
	span.AddAttributes(attrs...)

	defer func() {
		setSpanStatus(span, s.options, err)
		span.End()
	}()

	res, err = s.parent.Exec(args)
	if err != nil {
		return nil, err
	}

	res, err = ocResult{parent: res, ctx: ctx, options: s.options}, nil
	return
}

func (s ocStmt) Close() error {
	return s.parent.Close()
}

func (s ocStmt) NumInput() int {
	return s.parent.NumInput()
}

func (s ocStmt) Query(args []driver.Value) (rows driver.Rows, err error) {
	onDeferWithErr := recordCallStats(context.Background(), "go.sql.stmt.query", s.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	if !s.options.AllowRoot {
		return s.parent.Query(args)
	}

	ctx, span := trace.StartSpan(context.Background(), "sql:query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithSampler(s.options.Sampler),
	)
	attrs := make([]trace.Attribute, 0, len(s.options.DefaultAttributes)+2)
	attrs = append(attrs, s.options.DefaultAttributes...)
	attrs = append(
		attrs,
		attrDeprecated,
		trace.StringAttribute(
			"ocsql.deprecated", "driver does not support StmtQueryContext",
		),
	)
	if s.options.Query {
		attrs = append(attrs, trace.StringAttribute("sql.query", s.query))
		if s.options.QueryParams {
			attrs = append(attrs, paramsAttr(args)...)
		}
	}
	span.AddAttributes(attrs...)

	defer func() {
		setSpanStatus(span, s.options, err)
		span.End()
	}()

	rows, err = s.parent.Query(args)
	if err != nil {
		return nil, err
	}
	rows, err = wrapRows(ctx, rows, s.options), nil
	return
}

func (s ocStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (res driver.Result, err error) {
	onDeferWithErr := recordCallStats(ctx, "go.sql.stmt.exec", s.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	parentSpan := trace.FromContext(ctx)
	if !s.options.AllowRoot && parentSpan == nil {
		// we already tested driver to implement StmtExecContext
		return s.parent.(driver.StmtExecContext).ExecContext(ctx, args)
	}

	var span *trace.Span
	if parentSpan == nil {
		ctx, span = trace.StartSpan(ctx, "sql:exec",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(s.options.Sampler),
		)
	} else {
		_, span = trace.StartSpan(ctx, "sql:exec",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(s.options.Sampler),
		)
	}
	attrs := append([]trace.Attribute(nil), s.options.DefaultAttributes...)
	if s.options.Query {
		attrs = append(attrs, trace.StringAttribute("sql.query", s.query))
		if s.options.QueryParams {
			attrs = append(attrs, namedParamsAttr(args)...)
		}
	}
	span.AddAttributes(attrs...)

	defer func() {
		setSpanStatus(span, s.options, err)
		span.End()
	}()

	// we already tested driver to implement StmtExecContext
	execContext := s.parent.(driver.StmtExecContext)
	res, err = execContext.ExecContext(ctx, args)
	if err != nil {
		return nil, err
	}
	res, err = ocResult{parent: res, ctx: ctx, options: s.options}, nil
	return
}

func (s ocStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (rows driver.Rows, err error) {
	onDeferWithErr := recordCallStats(ctx, "go.sql.stmt.query", s.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	parentSpan := trace.FromContext(ctx)
	if !s.options.AllowRoot && parentSpan == nil {
		// we already tested driver to implement StmtQueryContext
		return s.parent.(driver.StmtQueryContext).QueryContext(ctx, args)
	}

	var span *trace.Span
	if parentSpan == nil {
		ctx, span = trace.StartSpan(ctx, "sql:query",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(s.options.Sampler),
		)
	} else {
		_, span = trace.StartSpan(ctx, "sql:query",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(s.options.Sampler),
		)
	}
	attrs := append([]trace.Attribute(nil), s.options.DefaultAttributes...)
	if s.options.Query {
		attrs = append(attrs, trace.StringAttribute("sql.query", s.query))
		if s.options.QueryParams {
			attrs = append(attrs, namedParamsAttr(args)...)
		}
	}
	span.AddAttributes(attrs...)

	defer func() {
		setSpanStatus(span, s.options, err)
		span.End()
	}()

	// we already tested driver to implement StmtQueryContext
	queryContext := s.parent.(driver.StmtQueryContext)
	rows, err = queryContext.QueryContext(ctx, args)
	if err != nil {
		return nil, err
	}
	rows, err = wrapRows(ctx, rows, s.options), nil
	return
}

// withRowsColumnTypeScanType is the same as the driver.RowsColumnTypeScanType
// interface except it omits the driver.Rows embedded interface.
// If the original driver.Rows implementation wrapped by ocsql supports
// RowsColumnTypeScanType we enable the original method implementation in the
// returned driver.Rows from wrapRows by doing a composition with ocRows.
type withRowsColumnTypeScanType interface {
	ColumnTypeScanType(index int) reflect.Type
}

// ocRows implements driver.Rows and all enhancement interfaces except
// driver.RowsColumnTypeScanType.
type ocRows struct {
	parent  driver.Rows
	ctx     context.Context
	options TraceOptions
}

// HasNextResultSet calls the implements the driver.RowsNextResultSet for ocRows.
// It returns the the underlying result of HasNextResultSet from the ocRows.parent
// if the parent implements driver.RowsNextResultSet.
func (r ocRows) HasNextResultSet() bool {
	if v, ok := r.parent.(driver.RowsNextResultSet); ok {
		return v.HasNextResultSet()
	}

	return false
}

// NextResultsSet calls the implements the driver.RowsNextResultSet for ocRows.
// It returns the the underlying result of NextResultSet from the ocRows.parent
// if the parent implements driver.RowsNextResultSet.
func (r ocRows) NextResultSet() error {
	if v, ok := r.parent.(driver.RowsNextResultSet); ok {
		return v.NextResultSet()
	}

	return io.EOF
}

// ColumnTypeDatabaseTypeName calls the implements the driver.RowsColumnTypeDatabaseTypeName for ocRows.
// It returns the the underlying result of ColumnTypeDatabaseTypeName from the ocRows.parent
// if the parent implements driver.RowsColumnTypeDatabaseTypeName.
func (r ocRows) ColumnTypeDatabaseTypeName(index int) string {
	if v, ok := r.parent.(driver.RowsColumnTypeDatabaseTypeName); ok {
		return v.ColumnTypeDatabaseTypeName(index)
	}

	return ""
}

// ColumnTypeLength calls the implements the driver.RowsColumnTypeLength for ocRows.
// It returns the the underlying result of ColumnTypeLength from the ocRows.parent
// if the parent implements driver.RowsColumnTypeLength.
func (r ocRows) ColumnTypeLength(index int) (length int64, ok bool) {
	if v, ok := r.parent.(driver.RowsColumnTypeLength); ok {
		return v.ColumnTypeLength(index)
	}

	return 0, false
}

// ColumnTypeNullable calls the implements the driver.RowsColumnTypeNullable for ocRows.
// It returns the the underlying result of ColumnTypeNullable from the ocRows.parent
// if the parent implements driver.RowsColumnTypeNullable.
func (r ocRows) ColumnTypeNullable(index int) (nullable, ok bool) {
	if v, ok := r.parent.(driver.RowsColumnTypeNullable); ok {
		return v.ColumnTypeNullable(index)
	}

	return false, false
}

// ColumnTypePrecisionScale calls the implements the driver.RowsColumnTypePrecisionScale for ocRows.
// It returns the the underlying result of ColumnTypePrecisionScale from the ocRows.parent
// if the parent implements driver.RowsColumnTypePrecisionScale.
func (r ocRows) ColumnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	if v, ok := r.parent.(driver.RowsColumnTypePrecisionScale); ok {
		return v.ColumnTypePrecisionScale(index)
	}

	return 0, 0, false
}

func (r ocRows) Columns() []string {
	return r.parent.Columns()
}

func (r ocRows) Close() (err error) {
	if r.options.RowsClose {
		_, span := trace.StartSpan(r.ctx, "sql:rows_close",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(r.options.Sampler),
		)
		if len(r.options.DefaultAttributes) > 0 {
			span.AddAttributes(r.options.DefaultAttributes...)
		}
		defer func() {
			setSpanStatus(span, r.options, err)
			span.End()
		}()
	}

	err = r.parent.Close()
	return
}

func (r ocRows) Next(dest []driver.Value) (err error) {
	if r.options.RowsNext {
		_, span := trace.StartSpan(r.ctx, "sql:rows_next",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(r.options.Sampler),
		)
		if len(r.options.DefaultAttributes) > 0 {
			span.AddAttributes(r.options.DefaultAttributes...)
		}
		defer func() {
			if err == io.EOF {
				// not an error; expected to happen during iteration
				setSpanStatus(span, r.options, nil)
			} else {
				setSpanStatus(span, r.options, err)
			}
			span.End()
		}()
	}

	err = r.parent.Next(dest)
	return
}

// wrapRows returns a struct which conforms to the driver.Rows interface.
// ocRows implements all enhancement interfaces that have no effect on
// sql/database logic in case the underlying parent implementation lacks them.
// Currently the one exception is RowsColumnTypeScanType which does not have a
// valid zero value. This interface is tested for and only enabled in case the
// parent implementation supports it.
func wrapRows(ctx context.Context, parent driver.Rows, options TraceOptions) driver.Rows {
	var (
		ts, hasColumnTypeScan = parent.(driver.RowsColumnTypeScanType)
	)

	r := ocRows{
		parent:  parent,
		ctx:     ctx,
		options: options,
	}

	if hasColumnTypeScan {
		return struct {
			ocRows
			withRowsColumnTypeScanType
		}{r, ts}
	}

	return r
}

// ocTx implements driver.Tx
type ocTx struct {
	parent  driver.Tx
	ctx     context.Context
	options TraceOptions
}

func (t ocTx) Commit() (err error) {
	onDeferWithErr := recordCallStats(context.Background(), "go.sql.commit", t.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	_, span := trace.StartSpan(t.ctx, "sql:commit",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithSampler(t.options.Sampler),
	)
	if len(t.options.DefaultAttributes) > 0 {
		span.AddAttributes(t.options.DefaultAttributes...)
	}
	defer func() {
		setSpanStatus(span, t.options, err)
		span.End()
	}()

	err = t.parent.Commit()
	return
}

func (t ocTx) Rollback() (err error) {
	onDeferWithErr := recordCallStats(context.Background(), "go.sql.rollback", t.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(err)
	}()

	_, span := trace.StartSpan(t.ctx, "sql:rollback",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithSampler(t.options.Sampler),
	)
	if len(t.options.DefaultAttributes) > 0 {
		span.AddAttributes(t.options.DefaultAttributes...)
	}
	defer func() {
		setSpanStatus(span, t.options, err)
		span.End()
	}()

	err = t.parent.Rollback()
	return
}

func paramsAttr(args []driver.Value) []trace.Attribute {
	attrs := make([]trace.Attribute, 0, len(args))
	for i, arg := range args {
		key := "sql.arg" + strconv.Itoa(i)
		attrs = append(attrs, argToAttr(key, arg))
	}
	return attrs
}

func namedParamsAttr(args []driver.NamedValue) []trace.Attribute {
	attrs := make([]trace.Attribute, 0, len(args))
	for _, arg := range args {
		var key string
		if arg.Name != "" {
			key = arg.Name
		} else {
			key = "sql.arg." + strconv.Itoa(arg.Ordinal)
		}
		attrs = append(attrs, argToAttr(key, arg.Value))
	}
	return attrs
}

func argToAttr(key string, val interface{}) trace.Attribute {
	switch v := val.(type) {
	case nil:
		return trace.StringAttribute(key, "")
	case int64:
		return trace.Int64Attribute(key, v)
	case float64:
		return trace.StringAttribute(key, fmt.Sprintf("%f", v))
	case bool:
		return trace.BoolAttribute(key, v)
	case []byte:
		if len(v) > 256 {
			v = v[0:256]
		}
		return trace.StringAttribute(key, fmt.Sprintf("%s", v))
	default:
		s := fmt.Sprintf("%v", v)
		if len(s) > 256 {
			s = s[0:256]
		}
		return trace.StringAttribute(key, s)
	}
}

func setSpanStatus(span *trace.Span, opts TraceOptions, err error) {
	var status trace.Status
	switch err {
	case nil:
		status.Code = trace.StatusCodeOK
		span.SetStatus(status)
		return
	case driver.ErrSkip:
		status.Code = trace.StatusCodeUnimplemented
		if opts.DisableErrSkip {
			// Suppress driver.ErrSkip since at runtime some drivers might not have
			// certain features, and an error would pollute many spans.
			status.Code = trace.StatusCodeOK
		}
	case context.Canceled:
		status.Code = trace.StatusCodeCancelled
	case context.DeadlineExceeded:
		status.Code = trace.StatusCodeDeadlineExceeded
	case sql.ErrNoRows:
		status.Code = trace.StatusCodeNotFound
	case sql.ErrTxDone, errConnDone:
		status.Code = trace.StatusCodeFailedPrecondition
	default:
		status.Code = trace.StatusCodeUnknown
	}
	status.Message = err.Error()
	span.SetStatus(status)
}
