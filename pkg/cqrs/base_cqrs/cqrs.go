package base_cqrs

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	sq "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	sqexp "github.com/doug-martin/goqu/v9/exp"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	sqlc_postgres "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/run"
	"github.com/inngest/inngest/pkg/util"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/jinzhu/copier"
	"github.com/oklog/ulid/v2"
)

const (
	forceHTTPS = false
)

var (
	// end represents a ulid ending with 'Z', eg. a far out cursor.
	endULID = ulid.ULID([16]byte{'Z'})
	nilULID = ulid.ULID{}
	nilUUID = uuid.UUID{}
)

func NewQueries(db *sql.DB, driver string) (q sqlc.Querier) {
	if driver == "postgres" {
		q = sqlc_postgres.NewNormalized(db)
	} else {
		q = sqlc.New(db)
	}

	return q
}

func NewCQRS(db *sql.DB, driver string) cqrs.Manager {
	return wrapper{
		driver: driver,
		q:      NewQueries(db, driver),
		db:     db,
	}
}

type wrapper struct {
	driver string
	q      sqlc.Querier
	db     *sql.DB
	tx     *sql.Tx
}

func (w wrapper) isPostgres() bool {
	return w.driver == "postgres"
}

func (w wrapper) dialect() string {
	if w.isPostgres() {
		return "postgres"
	}

	return "sqlite3"
}

// LoadFunction implements the state.FunctionLoader interface.
func (w wrapper) LoadFunction(ctx context.Context, envID, fnID uuid.UUID) (*state.ExecutorFunction, error) {
	// XXX: This doesn't store versions, as the dev server is currently ignorant to version.s
	fn, err := w.GetFunctionByInternalUUID(ctx, envID, fnID)
	if err != nil {
		return nil, err
	}
	def, err := fn.InngestFunction()
	if err != nil {
		return nil, err
	}

	return &state.ExecutorFunction{
		Function: def,
		Paused:   false, // dev server does not support pausing
	}, nil
}

func (w wrapper) WithTx(ctx context.Context) (cqrs.TxManager, error) {
	if w.tx != nil {
		// Already in a tx else DB would be present.
		return w, nil
	}
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	var q sqlc.Querier
	if w.isPostgres() {
		q = sqlc_postgres.NewNormalized(tx)
	} else {
		q = sqlc.New(tx)
	}

	return &wrapper{
		q:  q,
		tx: tx,
	}, nil
}

func (w wrapper) Commit(ctx context.Context) error {
	return w.tx.Commit()
}

func (w wrapper) Rollback(ctx context.Context) error {
	return w.tx.Rollback()
}

func (w wrapper) GetLatestQueueSnapshot(ctx context.Context) (*cqrs.QueueSnapshot, error) {
	chunks, err := w.q.GetLatestQueueSnapshotChunks(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting queue snapshot: %w", err)
	}

	var data []byte
	for _, chunk := range chunks {
		data = append(data, chunk.Data...)
	}

	if len(data) == 0 {
		return nil, nil
	}

	var snapshot cqrs.QueueSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("error unmarshalling queue snapshot: %w", err)
	}

	return &snapshot, nil
}

func (w wrapper) GetQueueSnapshot(ctx context.Context, snapshotID cqrs.SnapshotID) (*cqrs.QueueSnapshot, error) {
	chunks, err := w.q.GetQueueSnapshotChunks(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("error getting queue snapshot: %w", err)
	}

	var data []byte
	for _, chunk := range chunks {
		data = append(data, chunk.Data...)
	}

	if len(data) == 0 {
		return nil, nil
	}

	var snapshot cqrs.QueueSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("error unmarshalling queue snapshot: %w", err)
	}

	return &snapshot, nil
}

func (w wrapper) InsertQueueSnapshot(ctx context.Context, params cqrs.InsertQueueSnapshotParams) (cqrs.SnapshotID, error) {
	var snapshotID cqrs.SnapshotID

	byt, err := json.Marshal(params.Snapshot)
	if err != nil {
		return snapshotID, fmt.Errorf("error marshalling snapshot: %w", err)
	}

	var chunks [][]byte
	for len(byt) > 0 {
		if len(byt) > consts.StartMaxQueueChunkSize {
			chunks = append(chunks, byt[:consts.StartMaxQueueChunkSize])
			byt = byt[consts.StartMaxQueueChunkSize:]
		} else {
			chunks = append(chunks, byt)
			break
		}
	}

	snapshotID = ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)

	tx, err := w.WithTx(ctx)
	if err != nil {
		return snapshotID, fmt.Errorf("error starting transaction: %w", err)
	}

	// Insert each chunk of the snapshot.
	for i, chunk := range chunks {
		err = tx.InsertQueueSnapshotChunk(ctx, cqrs.InsertQueueSnapshotChunkParams{
			SnapshotID: snapshotID,
			ChunkID:    i,
			Chunk:      chunk,
		})
		if err != nil {
			return snapshotID, fmt.Errorf("error inserting queue snapshot chunk: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return snapshotID, fmt.Errorf("error committing transaction: %w", err)
	}

	// Asynchronously remove old snapshots.
	go func() {
		_, _ = w.q.DeleteOldQueueSnapshots(ctx, consts.StartMaxQueueSnapshots)
	}()

	return snapshotID, nil
}

func (w wrapper) InsertQueueSnapshotChunk(ctx context.Context, params cqrs.InsertQueueSnapshotChunkParams) error {
	err := w.q.InsertQueueSnapshotChunk(ctx, sqlc.InsertQueueSnapshotChunkParams{
		SnapshotID: params.SnapshotID.String(),
		ChunkID:    int64(params.ChunkID),
		Data:       params.Chunk,
	})
	if err != nil {
		return fmt.Errorf("error inserting queue snapshot chunk: %w", err)
	}

	return nil
}

//
// Apps
//

// GetApps returns apps that have not been deleted.
func (w wrapper) GetApps(ctx context.Context, envID uuid.UUID, filter *cqrs.FilterAppParam) ([]*cqrs.App, error) {
	f := func(ctx context.Context) ([]*sqlc.App, error) {
		return w.q.GetApps(ctx)
	}

	apps, err := copyInto(ctx, f, []*cqrs.App{})
	if err != nil {
		return nil, fmt.Errorf("could not get apps: %w", err)
	}

	filtered := make([]*cqrs.App, 0, len(apps))
	for _, app := range apps {
		if filter != nil && filter.ConnectionType != nil && filter.ConnectionType.String() != app.ConnectionType {
			continue
		}
		filtered = append(filtered, app)
	}

	return filtered, nil
}

func (w wrapper) GetAppByChecksum(ctx context.Context, envID uuid.UUID, checksum string) (*cqrs.App, error) {
	f := func(ctx context.Context) (*sqlc.App, error) {
		return w.q.GetAppByChecksum(ctx, checksum)
	}
	return copyInto(ctx, f, &cqrs.App{})
}

func (w wrapper) GetAppByID(ctx context.Context, id uuid.UUID) (*cqrs.App, error) {
	f := func(ctx context.Context) (*sqlc.App, error) {
		return w.q.GetAppByID(ctx, id)
	}
	return copyInto(ctx, f, &cqrs.App{})
}

func (w wrapper) GetAppByURL(ctx context.Context, envID uuid.UUID, url string) (*cqrs.App, error) {
	// Normalize the URL before inserting into the DB.
	url = util.NormalizeAppURL(url, forceHTTPS)

	f := func(ctx context.Context) (*sqlc.App, error) {
		return w.q.GetAppByURL(ctx, url)
	}
	return copyInto(ctx, f, &cqrs.App{})
}

func (w wrapper) GetAppByName(ctx context.Context, envID uuid.UUID, name string) (*cqrs.App, error) {
	f := func(ctx context.Context) (*sqlc.App, error) {
		return w.q.GetAppByName(ctx, name)
	}
	return copyInto(ctx, f, &cqrs.App{})
}

// GetAllApps returns all apps.
func (w wrapper) GetAllApps(ctx context.Context, envID uuid.UUID) ([]*cqrs.App, error) {
	return copyInto(ctx, w.q.GetAllApps, []*cqrs.App{})
}

// InsertApp creates a new app.
func (w wrapper) UpsertApp(ctx context.Context, arg cqrs.UpsertAppParams) (*cqrs.App, error) {
	// Normalize the URL before inserting into the DB.
	arg.Url = util.NormalizeAppURL(arg.Url, forceHTTPS)

	if arg.ConnectionType == "" {
		arg.ConnectionType = enums.AppConnectionTypeServerless.String()
	}

	return copyWriter(
		ctx,
		w.q.UpsertApp,
		arg,
		sqlc.UpsertAppParams{},
		&cqrs.App{},
	)
}

func (w wrapper) UpdateAppError(ctx context.Context, arg cqrs.UpdateAppErrorParams) (*cqrs.App, error) {
	// https://duckdb.org/docs/sql/indexes.html
	//
	// NOTE: You cannot update in DuckDB without deleting first right now.  Instead,
	// we run a series of transactions to get, delete, then re-insert the app.  This
	// will be fixed in a near version of DuckDB.
	app, err := w.q.GetApp(ctx, arg.ID)
	if err != nil {
		return nil, err
	}
	if err := w.q.DeleteApp(ctx, arg.ID); err != nil {
		return nil, err
	}

	app.Error = arg.Error
	params := sqlc.UpsertAppParams{}
	_ = copier.CopyWithOption(&params, app, copier.Option{DeepCopy: true})

	// Recreate the app.
	app, err = w.q.UpsertApp(ctx, params)
	if err != nil {
		return nil, err
	}
	out := &cqrs.App{}
	err = copier.CopyWithOption(out, app, copier.Option{DeepCopy: true})
	return out, err
}

func (w wrapper) UpdateAppURL(ctx context.Context, arg cqrs.UpdateAppURLParams) (*cqrs.App, error) {
	// Normalize the URL before inserting into the DB.
	arg.Url = util.NormalizeAppURL(arg.Url, forceHTTPS)

	// https://duckdb.org/docs/sql/indexes.html
	//
	// NOTE: You cannot update in DuckDB without deleting first right now.  Instead,
	// we run a series of transactions to get, delete, then re-insert the app.  This
	// will be fixed in a near version of DuckDB.
	app, err := w.q.GetApp(ctx, arg.ID)
	if err != nil {
		return nil, err
	}
	if err := w.q.DeleteApp(ctx, arg.ID); err != nil {
		return nil, err
	}
	app.Url = arg.Url
	params := sqlc.UpsertAppParams{}
	_ = copier.CopyWithOption(&params, app, copier.Option{DeepCopy: true})
	// Recreate the app.
	app, err = w.q.UpsertApp(ctx, params)
	if err != nil {
		return nil, err
	}
	out := &cqrs.App{}
	err = copier.CopyWithOption(out, app, copier.Option{DeepCopy: true})
	return out, err
}

// DeleteApp deletes an app
func (w wrapper) DeleteApp(ctx context.Context, id uuid.UUID) error {
	return w.q.DeleteApp(ctx, id)
}

//
// Functions
//

func (w wrapper) GetAppFunctions(ctx context.Context, appID uuid.UUID) ([]*cqrs.Function, error) {
	f := func(ctx context.Context) ([]*sqlc.Function, error) {
		return w.q.GetAppFunctions(ctx, appID)
	}
	return copyInto(ctx, f, []*cqrs.Function{})
}

func (w wrapper) GetFunctionByExternalID(ctx context.Context, wsID uuid.UUID, appID, fnSlug string) (*cqrs.Function, error) {
	f := func(ctx context.Context) (*sqlc.Function, error) {
		return w.q.GetFunctionBySlug(ctx, fnSlug)
	}
	return copyInto(ctx, f, &cqrs.Function{})
}

func (w wrapper) GetFunctionByInternalUUID(ctx context.Context, wsID, fnID uuid.UUID) (*cqrs.Function, error) {
	f := func(ctx context.Context) (*sqlc.Function, error) {
		return w.q.GetFunctionByID(ctx, fnID)
	}
	return copyInto(ctx, f, &cqrs.Function{})
}

func (w wrapper) GetFunctions(ctx context.Context) ([]*cqrs.Function, error) {
	return copyInto(ctx, w.q.GetFunctions, []*cqrs.Function{})
}

func (w wrapper) GetFunctionsByAppInternalID(ctx context.Context, workspaceID, appID uuid.UUID) ([]*cqrs.Function, error) {
	f := func(ctx context.Context) ([]*sqlc.Function, error) {
		// Ingore the workspace ID for now.
		return w.q.GetAppFunctions(ctx, appID)
	}
	return copyInto(ctx, f, []*cqrs.Function{})
}

func (w wrapper) GetFunctionsByAppExternalID(ctx context.Context, workspaceID uuid.UUID, appID string) ([]*cqrs.Function, error) {
	f := func(ctx context.Context) ([]*sqlc.Function, error) {
		// Ingore the workspace ID for now.
		return w.q.GetAppFunctionsBySlug(ctx, appID)
	}
	return copyInto(ctx, f, []*cqrs.Function{})
}

func (w wrapper) InsertFunction(ctx context.Context, params cqrs.InsertFunctionParams) (*cqrs.Function, error) {
	return copyWriter(
		ctx,
		w.q.InsertFunction,
		params,
		sqlc.InsertFunctionParams{},
		&cqrs.Function{},
	)
}

func (w wrapper) DeleteFunctionsByAppID(ctx context.Context, appID uuid.UUID) error {
	return w.q.DeleteFunctionsByAppID(ctx, appID)
}

func (w wrapper) DeleteFunctionsByIDs(ctx context.Context, ids []uuid.UUID) error {
	return w.q.DeleteFunctionsByIDs(ctx, ids)
}

func (w wrapper) UpdateFunctionConfig(ctx context.Context, arg cqrs.UpdateFunctionConfigParams) (*cqrs.Function, error) {
	return copyWriter(
		ctx,
		w.q.UpdateFunctionConfig,
		arg,
		sqlc.UpdateFunctionConfigParams{},
		&cqrs.Function{},
	)
}

//
// Events
//

func (w wrapper) InsertEvent(ctx context.Context, e cqrs.Event) error {
	data, err := json.Marshal(e.EventData)
	if err != nil {
		return err
	}
	user, err := json.Marshal(e.EventUser)
	if err != nil {
		return err
	}
	evt := sqlc.InsertEventParams{
		InternalID: e.ID,
		ReceivedAt: time.Now(),
		EventID:    e.EventID,
		EventName:  e.EventName,
		EventData:  string(data),
		EventUser:  string(user),
		EventV: sql.NullString{
			Valid:  e.EventVersion != "",
			String: e.EventVersion,
		},
		EventTs: time.UnixMilli(e.EventTS),
	}
	return w.q.InsertEvent(ctx, evt)
}

func (w wrapper) InsertEventBatch(ctx context.Context, eb cqrs.EventBatch) error {
	evtIDs := make([]string, len(eb.Events))
	for i, evt := range eb.Events {
		evtIDs[i] = evt.GetInternalID().String()
	}

	batch := sqlc.InsertEventBatchParams{
		ID:          eb.ID,
		AccountID:   eb.AccountID,
		WorkspaceID: eb.WorkspaceID,
		AppID:       eb.AppID,
		WorkflowID:  eb.FunctionID,
		RunID:       eb.RunID,
		StartedAt:   eb.StartedAt(),
		ExecutedAt:  eb.ExecutedAt(),
		EventIds:    []byte(strings.Join(evtIDs, ",")),
	}

	return w.q.InsertEventBatch(ctx, batch)
}

func (w wrapper) GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*cqrs.Event, error) {
	obj, err := w.q.GetEventByInternalID(ctx, internalID)
	if err != nil {
		return nil, err
	}
	evt := convertEvent(obj)
	return &evt, nil
}

func (w wrapper) GetEventBatchesByEventID(ctx context.Context, eventID ulid.ULID) ([]*cqrs.EventBatch, error) {
	batches, err := w.q.GetEventBatchesByEventID(ctx, eventID.String())
	if err != nil {
		return nil, err
	}

	var out = make([]*cqrs.EventBatch, len(batches))
	for n, i := range batches {
		eb := convertEventBatch(i)
		out[n] = &eb
	}

	return out, nil
}
func (w wrapper) GetEventBatchByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.EventBatch, error) {
	obj, err := w.q.GetEventBatchByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}

	eb := convertEventBatch(obj)
	return &eb, nil
}

func (w wrapper) GetEventsByInternalIDs(ctx context.Context, ids []ulid.ULID) ([]*cqrs.Event, error) {
	objs, err := w.q.GetEventsByInternalIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	evts := make([]*cqrs.Event, len(objs))
	for i, o := range objs {
		evt := convertEvent(o)
		evts[i] = &evt
	}

	return evts, nil
}

func (w wrapper) GetEventsByExpressions(ctx context.Context, cel []string) ([]*cqrs.Event, error) {
	expHandler, err := run.NewExpressionHandler(ctx,
		run.WithExpressionHandlerExpressions(cel),
	)
	if err != nil {
		return nil, err
	}
	prefilters, err := expHandler.ToSQLFilters(ctx)
	if err != nil {
		return nil, err
	}

	sql, args, err := sq.Dialect(w.dialect()).
		From("events").
		Select(
			"internal_id",
			"account_id",
			"workspace_id",
			"source",
			"source_id",
			"received_at",
			"event_id",
			"event_name",
			"event_data",
			"event_user",
			"event_v",
			"event_ts",
		).
		Where(prefilters...).
		Order(sq.C("received_at").Desc()).
		ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := w.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	res := []*cqrs.Event{}
	for rows.Next() {
		data := sqlc.Event{}
		if err := rows.Scan(
			&data.InternalID,
			&data.AccountID,
			&data.WorkspaceID,
			&data.Source,
			&data.SourceID,
			&data.ReceivedAt,
			&data.EventID,
			&data.EventName,
			&data.EventData,
			&data.EventUser,
			&data.EventV,
			&data.EventTs,
		); err != nil {
			return nil, err
		}

		evt, err := data.ToCQRS()
		if err != nil {
			return nil, fmt.Errorf("error deserializing event: %w", err)
		}

		ok, err := expHandler.MatchEventExpressions(ctx, evt.GetEvent())
		if err != nil {
			return nil, err
		}
		if ok {
			res = append(res, evt)
		}
	}

	return res, nil
}

func (w wrapper) FindEvent(ctx context.Context, workspaceID uuid.UUID, internalID ulid.ULID) (*cqrs.Event, error) {
	return w.GetEventByInternalID(ctx, internalID)
}

func (w wrapper) WorkspaceEvents(ctx context.Context, workspaceID uuid.UUID, opts *cqrs.WorkspaceEventsOpts) ([]cqrs.Event, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	if opts.Cursor == nil {
		opts.Cursor = &endULID
	}

	var (
		evts []*sqlc.Event
		err  error
	)

	if opts.Name == nil {
		params := sqlc.WorkspaceEventsParams{
			Cursor: *opts.Cursor,
			Before: opts.Newest,
			After:  opts.Oldest,
			Limit:  int64(opts.Limit),
		}
		evts, err = w.q.WorkspaceEvents(ctx, params)
	} else {
		params := sqlc.WorkspaceNamedEventsParams{
			Name:   *opts.Name,
			Cursor: *opts.Cursor,
			Before: opts.Newest,
			After:  opts.Oldest,
			Limit:  int64(opts.Limit),
		}
		evts, err = w.q.WorkspaceNamedEvents(ctx, params)
	}

	if err != nil {
		return nil, err
	}
	out := make([]cqrs.Event, len(evts))
	for n, evt := range evts {
		out[n] = convertEvent(evt)
	}
	return out, nil
}

func (w wrapper) GetEventsIDbound(
	ctx context.Context,
	ids cqrs.IDBound,
	limit int,
	includeInternal bool,
) ([]*cqrs.Event, error) {

	if ids.Before == nil {
		ids.Before = &endULID
	}

	if ids.After == nil {
		ids.After = &nilULID
	}

	evts, err := w.q.GetEventsIDbound(ctx, sqlc.GetEventsIDboundParams{
		After:           *ids.After,
		Before:          *ids.Before,
		IncludeInternal: strconv.FormatBool(includeInternal),
		Limit:           int64(limit),
	})
	if err != nil {
		return []*cqrs.Event{}, err
	}

	var res = make([]*cqrs.Event, len(evts))
	for n, i := range evts {
		e := convertEvent(i)
		res[n] = &e
	}
	return res, nil
}

func convertEvent(obj *sqlc.Event) cqrs.Event {
	evt := &cqrs.Event{
		ID:           obj.InternalID,
		ReceivedAt:   obj.ReceivedAt,
		EventID:      obj.EventID,
		EventName:    obj.EventName,
		EventVersion: obj.EventV.String,
		EventTS:      obj.EventTs.UnixMilli(),
		EventData:    map[string]any{},
		EventUser:    map[string]any{},
	}
	_ = json.Unmarshal([]byte(obj.EventData), &evt.EventData)
	_ = json.Unmarshal([]byte(obj.EventUser), &evt.EventUser)
	return *evt
}

func convertEventBatch(obj *sqlc.EventBatch) cqrs.EventBatch {
	var evtIDs []ulid.ULID
	if ids, err := obj.EventIDs(); err == nil {
		evtIDs = ids
	}

	eb := cqrs.NewEventBatch(
		cqrs.WithEventBatchID(obj.ID),
		cqrs.WithEventBatchAccountID(obj.AccountID),
		cqrs.WithEventBatchWorkspaceID(obj.WorkspaceID),
		cqrs.WithEventBatchAppID(obj.AppID),
		cqrs.WithEventBatchRunID(obj.RunID),
		cqrs.WithEventBatchEventIDs(evtIDs),
		cqrs.WithEventBatchExecutedTime(obj.ExecutedAt),
	)

	return *eb
}

//
// Function runs
//

func (w wrapper) InsertFunctionRun(ctx context.Context, e cqrs.FunctionRun) error {
	run := sqlc.InsertFunctionRunParams{}
	if err := copier.CopyWithOption(&run, e, copier.Option{DeepCopy: true}); err != nil {
		return err
	}

	// Need to manually set the cron field since `copier` won't do it.
	if e.Cron != nil {
		run.Cron = sql.NullString{
			Valid:  true,
			String: *e.Cron,
		}
	}

	return w.q.InsertFunctionRun(ctx, run)
}

func (w wrapper) GetFunctionRunsFromEvents(
	ctx context.Context,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
	eventIDs []ulid.ULID,
) ([]*cqrs.FunctionRun, error) {
	runs, err := w.q.GetFunctionRunsFromEvents(ctx, eventIDs)
	if err != nil {
		return nil, err
	}
	result := []*cqrs.FunctionRun{}
	for _, item := range runs {
		result = append(result, toCQRSRun(item.FunctionRun, item.FunctionFinish))
	}
	return result, nil
}

func (w wrapper) GetFunctionRun(
	ctx context.Context,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
	id ulid.ULID,
) (*cqrs.FunctionRun, error) {
	item, err := w.q.GetFunctionRun(ctx, id)
	if err != nil {
		return nil, err
	}
	return toCQRSRun(item.FunctionRun, item.FunctionFinish), nil
}

func (w wrapper) GetFunctionRunsTimebound(ctx context.Context, t cqrs.Timebound, limit int) ([]*cqrs.FunctionRun, error) {
	after := time.Time{}                           // after the beginning of time, eg all
	before := time.Now().Add(time.Hour * 24 * 365) // before 1 year in the future, eg all
	if t.After != nil {
		after = *t.After
	}
	if t.Before != nil {
		before = *t.Before
	}

	runs, err := w.q.GetFunctionRunsTimebound(ctx, sqlc.GetFunctionRunsTimeboundParams{
		Before: before,
		After:  after,
		Limit:  int64(limit),
	})
	if err != nil {
		return nil, err
	}
	result := []*cqrs.FunctionRun{}
	for _, item := range runs {
		result = append(result, toCQRSRun(item.FunctionRun, item.FunctionFinish))
	}
	return result, nil
}

func (w wrapper) GetFunctionRunFinishesByRunIDs(
	ctx context.Context,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
	runIDs []ulid.ULID,
) ([]*cqrs.FunctionRunFinish, error) {
	return copyInto(ctx, func(ctx context.Context) ([]*sqlc.FunctionFinish, error) {
		return w.q.GetFunctionRunFinishesByRunIDs(ctx, runIDs)
	}, []*cqrs.FunctionRunFinish{})
}

//
// History
//

func (w wrapper) InsertHistory(ctx context.Context, h history.History) error {
	params, err := convertHistoryToWriter(h)
	if err != nil {
		return err
	}
	return w.q.InsertHistory(ctx, *params)
}

func (w wrapper) GetFunctionRunHistory(ctx context.Context, runID ulid.ULID) ([]*history.History, error) {
	_, err := w.q.GetFunctionRunHistory(ctx, runID)
	// TODO: Convert history
	return nil, err
}

func toCQRSRun(run sqlc.FunctionRun, finish sqlc.FunctionFinish) *cqrs.FunctionRun {
	copied := cqrs.FunctionRun{
		RunID:           run.RunID,
		RunStartedAt:    run.RunStartedAt,
		FunctionID:      run.FunctionID,
		FunctionVersion: run.FunctionVersion,
		EventID:         run.EventID,
		WorkspaceID:     run.WorkspaceID,
	}
	if run.BatchID != nilULID {
		copied.BatchID = &run.BatchID
	}
	if run.OriginalRunID != nilULID {
		copied.OriginalRunID = &run.OriginalRunID
	}
	if run.Cron.Valid {
		copied.Cron = &run.Cron.String
	}
	if finish.Status.Valid {
		copied.Status, _ = enums.RunStatusString(finish.Status.String)
		copied.Output = json.RawMessage(finish.Output.String)
		copied.EndedAt = &finish.CreatedAt.Time
	}
	return &copied
}

//
// Trace
//

func (w wrapper) InsertSpan(ctx context.Context, span *cqrs.Span) error {
	params := &sqlc.InsertTraceParams{
		Timestamp:       span.Timestamp,
		TimestampUnixMs: span.Timestamp.UnixMilli(),
		TraceID:         span.TraceID,
		SpanID:          span.SpanID,
		SpanName:        span.SpanName,
		SpanKind:        span.SpanKind,
		ServiceName:     span.ServiceName,
		ScopeName:       span.ScopeName,
		ScopeVersion:    span.ScopeVersion,
		Duration:        int64(span.Duration / time.Millisecond),
		StatusCode:      span.StatusCode,
	}

	if span.RunID != nil {
		params.RunID = *span.RunID
	}
	if span.ParentSpanID != nil {
		params.ParentSpanID = sql.NullString{String: *span.ParentSpanID, Valid: true}
	}
	if span.TraceState != nil {
		params.TraceState = sql.NullString{String: *span.TraceState, Valid: true}
	}
	if byt, err := json.Marshal(span.ResourceAttributes); err == nil {
		params.ResourceAttributes = byt
	}
	if byt, err := json.Marshal(span.SpanAttributes); err == nil {
		params.SpanAttributes = byt
	}
	if byt, err := json.Marshal(span.Events); err == nil {
		params.Events = byt
	}
	if byt, err := json.Marshal(span.Links); err == nil {
		params.Links = byt
	}
	if span.StatusMessage != nil {
		params.StatusMessage = sql.NullString{String: *span.StatusMessage, Valid: true}
	}

	return w.q.InsertTrace(ctx, *params)
}

func (w wrapper) InsertTraceRun(ctx context.Context, run *cqrs.TraceRun) error {
	runid, err := ulid.Parse(run.RunID)
	if err != nil {
		return fmt.Errorf("error parsing runID as ULID: %w", err)
	}

	params := sqlc.InsertTraceRunParams{
		AccountID:   run.AccountID,
		WorkspaceID: run.WorkspaceID,
		AppID:       run.AppID,
		FunctionID:  run.FunctionID,
		TraceID:     []byte(run.TraceID),
		SourceID:    run.SourceID,
		RunID:       runid,
		QueuedAt:    run.QueuedAt.UnixMilli(),
		StartedAt:   run.StartedAt.UnixMilli(),
		EndedAt:     run.EndedAt.UnixMilli(),
		Status:      run.Status.ToCode(),
		TriggerIds:  []byte{},
		Output:      run.Output,
		IsDebounce:  run.IsDebounce,
		HasAi:       run.HasAI,
	}

	if run.BatchID != nil {
		params.BatchID = *run.BatchID
	}
	if run.CronSchedule != nil {
		params.CronSchedule = sql.NullString{String: *run.CronSchedule, Valid: true}
	}
	if len(run.TriggerIDs) > 0 {
		params.TriggerIds = []byte(strings.Join(run.TriggerIDs, ","))
	}

	return w.q.InsertTraceRun(ctx, params)
}

type traceRunCursorFilter struct {
	ID    string
	Value int64
}

func (w wrapper) GetTraceSpansByRun(ctx context.Context, id cqrs.TraceRunIdentifier) ([]*cqrs.Span, error) {
	spans, err := w.q.GetTraceSpans(ctx, sqlc.GetTraceSpansParams{
		TraceID: id.TraceID,
		RunID:   id.RunID,
	})
	if err != nil {
		return nil, err
	}

	res := []*cqrs.Span{}
	seen := map[string]bool{}
	for _, s := range spans {
		// identifier to used for checking if this span is seen already
		m := map[string]any{
			"ts":  s.Timestamp.UnixMilli(),
			"tid": s.TraceID,
			"sid": s.SpanID,
		}
		byt, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		ident := base64.StdEncoding.EncodeToString(byt)
		if _, ok := seen[ident]; ok {
			// already seen, so continue
			continue
		}

		span := &cqrs.Span{
			Timestamp:    s.Timestamp,
			TraceID:      string(s.TraceID),
			SpanID:       string(s.SpanID),
			SpanName:     s.SpanName,
			SpanKind:     s.SpanKind,
			ServiceName:  s.ServiceName,
			ScopeName:    s.ScopeName,
			ScopeVersion: s.ScopeVersion,
			Duration:     time.Duration(s.Duration * int64(time.Millisecond)),
			StatusCode:   s.StatusCode,
			RunID:        &s.RunID,
		}

		if s.StatusMessage.Valid {
			span.StatusMessage = &s.StatusMessage.String
		}

		if s.ParentSpanID.Valid {
			span.ParentSpanID = &s.ParentSpanID.String
		}
		if s.TraceState.Valid {
			span.TraceState = &s.TraceState.String
		}

		var resourceAttr, spanAttr map[string]string
		if err := json.Unmarshal(s.ResourceAttributes, &resourceAttr); err == nil {
			span.ResourceAttributes = resourceAttr
		}
		if err := json.Unmarshal(s.SpanAttributes, &spanAttr); err == nil {
			span.SpanAttributes = spanAttr
		}

		res = append(res, span)
		seen[ident] = true
	}

	return res, nil
}

func (w wrapper) FindOrBuildTraceRun(ctx context.Context, opts cqrs.FindOrCreateTraceRunOpt) (*cqrs.TraceRun, error) {
	run, err := w.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: opts.RunID})
	if err == nil {
		return run, nil
	}

	new := cqrs.TraceRun{
		AccountID:   opts.AccountID,
		WorkspaceID: opts.WorkspaceID,
		AppID:       opts.AppID,
		FunctionID:  opts.FunctionID,
		RunID:       opts.RunID.String(),
		TraceID:     opts.TraceID,
		QueuedAt:    ulid.Time(opts.RunID.Time()),
		TriggerIDs:  []string{},
		Status:      enums.RunStatusUnknown,
	}

	return &new, nil
}

func (w wrapper) GetTraceRun(ctx context.Context, id cqrs.TraceRunIdentifier) (*cqrs.TraceRun, error) {

	run, err := w.q.GetTraceRun(ctx, id.RunID)
	if err != nil {
		return nil, err
	}

	start := time.UnixMilli(run.StartedAt)
	end := time.UnixMilli(run.EndedAt)
	triggerIDS := strings.Split(string(run.TriggerIds), ",")

	var (
		isBatch bool
		batchID *ulid.ULID
		cron    *string
	)

	if run.BatchID != nilULID {
		isBatch = true
		batchID = &run.BatchID
	}

	if run.CronSchedule.Valid {
		cron = &run.CronSchedule.String
	}

	trun := cqrs.TraceRun{
		AccountID:    run.AccountID,
		WorkspaceID:  run.WorkspaceID,
		AppID:        run.AppID,
		FunctionID:   run.FunctionID,
		TraceID:      string(run.TraceID),
		RunID:        id.RunID.String(),
		QueuedAt:     time.UnixMilli(run.QueuedAt),
		StartedAt:    start,
		EndedAt:      end,
		Duration:     end.Sub(start),
		SourceID:     run.SourceID,
		TriggerIDs:   triggerIDS,
		Output:       run.Output,
		Status:       enums.RunCodeToStatus(run.Status),
		BatchID:      batchID,
		IsBatch:      isBatch,
		CronSchedule: cron,
		HasAI:        run.HasAi,
	}

	return &trun, nil
}

func (w wrapper) GetSpanOutput(ctx context.Context, opts cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	if opts.TraceID == "" {
		return nil, fmt.Errorf("traceID is required to retrieve output")
	}
	if opts.SpanID == "" {
		return nil, fmt.Errorf("spanID is required to retrieve output")
	}

	// query spans in descending order
	spans, err := w.q.GetTraceSpanOutput(ctx, sqlc.GetTraceSpanOutputParams{
		TraceID: opts.TraceID,
		SpanID:  opts.SpanID,
	})
	if err != nil {
		return nil, fmt.Errorf("error retrieving spans for output: %w", err)
	}

	for _, s := range spans {
		var evts []cqrs.SpanEvent
		err := json.Unmarshal(s.Events, &evts)
		if err != nil {
			return nil, fmt.Errorf("error parsing span outputs: %w", err)
		}

		var (
			input      []byte
			spanOutput *cqrs.SpanOutput
		)

		for _, evt := range evts {
			if spanOutput == nil {
				_, isFnOutput := evt.Attributes[consts.OtelSysFunctionOutput]
				_, isStepOutput := evt.Attributes[consts.OtelSysStepOutput]
				if isFnOutput || isStepOutput {
					var isError bool
					switch strings.ToUpper(s.StatusCode) {
					case "ERROR", "STATUS_CODE_ERROR":
						isError = true
					}

					spanOutput = &cqrs.SpanOutput{
						Data:             []byte(evt.Name),
						Timestamp:        evt.Timestamp,
						Attributes:       evt.Attributes,
						IsError:          isError,
						IsFunctionOutput: isFnOutput,
						IsStepOutput:     isStepOutput,
					}
				}
			}

			if _, isInput := evt.Attributes[consts.OtelSysStepInput]; isInput && input == nil {
				input = []byte(evt.Name)
			}

			if spanOutput != nil && input != nil {
				break
			}
		}

		if spanOutput != nil {
			spanOutput.Input = input
			return spanOutput, nil
		}
	}

	return nil, fmt.Errorf("no output found")
}

func (w wrapper) GetSpanStack(ctx context.Context, opts cqrs.SpanIdentifier) ([]string, error) {
	if opts.TraceID == "" {
		return nil, fmt.Errorf("traceID is required to retrieve stack")
	}
	if opts.SpanID == "" {
		return nil, fmt.Errorf("spanID is required to retrieve stack")
	}

	// query spans in descending order
	spans, err := w.q.GetTraceSpanOutput(ctx, sqlc.GetTraceSpanOutputParams{
		TraceID: opts.TraceID,
		SpanID:  opts.SpanID,
	})
	if err != nil {
		return nil, fmt.Errorf("error retrieving spans for stack: %w", err)
	}

	for _, s := range spans {
		var evts []cqrs.SpanEvent
		err := json.Unmarshal(s.Events, &evts)
		if err != nil {
			return nil, fmt.Errorf("error parsing span outputs: %w", err)
		}

		for _, evt := range evts {
			if _, isStackEvt := evt.Attributes[consts.OtelSysStepStack]; isStackEvt {
				// Data is kept in the `Name` field
				return strings.Split(evt.Name, ","), nil
			}
		}
	}

	return nil, fmt.Errorf("no stack found")
}

type runsQueryBuilder struct {
	filter       []sq.Expression
	order        []sqexp.OrderedExpression
	cursor       *cqrs.TracePageCursor
	cursorLayout *cqrs.TracePageCursor
}

func newRunsQueryBuilder(ctx context.Context, opt cqrs.GetTraceRunOpt) *runsQueryBuilder {
	// filters
	filter := []sq.Expression{}
	if len(opt.Filter.AppID) > 0 {
		filter = append(filter, sq.C("app_id").In(opt.Filter.AppID))
	}
	if len(opt.Filter.FunctionID) > 0 {
		filter = append(filter, sq.C("function_id").In(opt.Filter.FunctionID))
	}
	if len(opt.Filter.Status) > 0 {
		status := []int64{}
		for _, s := range opt.Filter.Status {
			switch s {
			case enums.RunStatusUnknown, enums.RunStatusOverflowed:
				continue
			}
			status = append(status, s.ToCode())
		}
		filter = append(filter, sq.C("status").In(status))
	}
	tsfield := strings.ToLower(opt.Filter.TimeField.String())
	filter = append(filter, sq.C(tsfield).Gte(opt.Filter.From.UnixMilli()))

	until := opt.Filter.Until
	if until.UnixMilli() <= 0 {
		until = time.Now()
	}
	filter = append(filter, sq.C(tsfield).Lt(until.UnixMilli()))

	// Layout to be used for the response cursors
	resCursorLayout := cqrs.TracePageCursor{
		Cursors: map[string]cqrs.TraceCursor{},
	}

	reqcursor := &cqrs.TracePageCursor{}
	if opt.Cursor != "" {
		if err := reqcursor.Decode(opt.Cursor); err != nil {
			log.From(ctx).Error().Err(err).Str("cursor", opt.Cursor).Msg("error decoding function run cursor")
		}
	}

	// order by
	//
	// When going through the sorting fields, construct
	// - response pagination cursor layout
	// - update filter with op against sorted fields for pagination
	sortOrder := []enums.TraceRunTime{}
	sortDir := map[enums.TraceRunTime]enums.TraceRunOrder{}
	cursorFilter := map[enums.TraceRunTime]traceRunCursorFilter{}
	for _, f := range opt.Order {
		sortDir[f.Field] = f.Direction
		found := false
		for _, field := range sortOrder {
			if f.Field == field {
				found = true
				break
			}
		}
		if !found {
			sortOrder = append(sortOrder, f.Field)
		}

		rc := reqcursor.Find(f.Field.String())
		if rc != nil {
			cursorFilter[f.Field] = traceRunCursorFilter{ID: reqcursor.ID, Value: rc.Value}
		}
		resCursorLayout.Add(f.Field.String())
	}

	order := []sqexp.OrderedExpression{}
	for _, f := range sortOrder {
		var o sqexp.OrderedExpression
		field := strings.ToLower(f.String())
		if d, ok := sortDir[f]; ok {
			switch d {
			case enums.TraceRunOrderAsc:
				o = sq.C(field).Asc()
			case enums.TraceRunOrderDesc:
				o = sq.C(field).Desc()
			default:
				log.From(ctx).Error().Str("field", field).Str("direction", d.String()).Msg("invalid direction specified for sorting")
				continue
			}

			order = append(order, o)
		}
	}
	order = append(order, sq.C("run_id").Asc())

	// cursor filter
	for k, cf := range cursorFilter {
		ord, ok := sortDir[k]
		if !ok {
			continue
		}

		var compare sq.Expression
		field := strings.ToLower(k.String())
		switch ord {
		case enums.TraceRunOrderAsc:
			compare = sq.C(field).Gt(cf.Value)
		case enums.TraceRunOrderDesc:
			compare = sq.C(field).Lt(cf.Value)
		default:
			continue
		}

		filter = append(filter, sq.Or(
			compare,
			sq.And(
				sq.C(field).Eq(cf.Value),
				sq.C("run_id").Gt(cf.ID),
			),
		))
	}

	return &runsQueryBuilder{
		filter:       filter,
		order:        order,
		cursor:       reqcursor,
		cursorLayout: &resCursorLayout,
	}
}

func (w wrapper) GetTraceRunsCount(ctx context.Context, opt cqrs.GetTraceRunOpt) (int, error) {
	// explicitly set it to zero so it would not attempt to paginate
	opt.Items = 0
	res, err := w.GetTraceRuns(ctx, opt)
	if err != nil {
		return 0, err
	}

	return len(res), nil
}

func (w wrapper) GetTraceRuns(ctx context.Context, opt cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
	// use evtIDs as post query filter
	evtIDs := []string{}
	expHandler, err := run.NewExpressionHandler(ctx,
		run.WithExpressionHandlerBlob(opt.Filter.CEL, "\n"),
	)
	if err != nil {
		return nil, err
	}
	if expHandler.HasEventFilters() {
		evts, err := w.GetEventsByExpressions(ctx, expHandler.EventExprList)
		if err != nil {
			return nil, err
		}
		for _, e := range evts {
			evtIDs = append(evtIDs, e.ID.String())
		}
	}

	builder := newRunsQueryBuilder(ctx, opt)
	filter := builder.filter
	order := builder.order
	reqcursor := builder.cursor
	resCursorLayout := builder.cursorLayout

	// read from database
	// TODO:
	// change this to a continuous loop with limits instead of just attempting to grab everything.
	// might not matter though since this is primarily meant for local
	// development
	sql, args, err := sq.Dialect(w.dialect()).
		From("trace_runs").
		Select(
			"app_id",
			"function_id",
			"trace_id",
			"run_id",
			"queued_at",
			"started_at",
			"ended_at",
			"status",
			"source_id",
			"trigger_ids",
			"output",
			"batch_id",
			"is_debounce",
			"cron_schedule",
			"has_ai",
		).
		Where(filter...).
		Order(order...).
		ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := w.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	res := []*cqrs.TraceRun{}
	var count uint
	for rows.Next() {
		data := sqlc.TraceRun{}
		err := rows.Scan(
			&data.AppID,
			&data.FunctionID,
			&data.TraceID,
			&data.RunID,
			&data.QueuedAt,
			&data.StartedAt,
			&data.EndedAt,
			&data.Status,
			&data.SourceID,
			&data.TriggerIds,
			&data.Output,
			&data.BatchID,
			&data.IsDebounce,
			&data.CronSchedule,
			&data.HasAi,
		)
		if err != nil {
			return nil, err
		}

		// filter out runs that doesn't have the event IDs
		if len(evtIDs) > 0 && !data.HasEventIDs(evtIDs) {
			continue
		}

		// the cursor target should be skipped
		if reqcursor.ID == data.RunID.String() {
			continue
		}

		if expHandler.HasOutputFilters() {
			ok, err := expHandler.MatchOutputExpressions(ctx, data.Output)
			if err != nil {
				logger.StdlibLogger(ctx).Error("error inspecting run for output match",
					"error", err,
					"output", string(data.Output),
					"acctID", data.AccountID,
					"wsID", data.WorkspaceID,
					"appID", data.AppID,
					"wfID", data.FunctionID,
					"runID", data.RunID,
				)
				continue
			}
			if !ok {
				continue
			}
		}

		// copy layout
		pc := resCursorLayout
		// construct the needed fields to generate a cursor representing this run
		pc.ID = data.RunID.String()
		for k := range pc.Cursors {
			switch k {
			case strings.ToLower(enums.TraceRunTimeQueuedAt.String()):
				pc.Cursors[k] = cqrs.TraceCursor{Field: k, Value: data.QueuedAt}
			case strings.ToLower(enums.TraceRunTimeStartedAt.String()):
				pc.Cursors[k] = cqrs.TraceCursor{Field: k, Value: data.StartedAt}
			case strings.ToLower(enums.TraceRunTimeEndedAt.String()):
				pc.Cursors[k] = cqrs.TraceCursor{Field: k, Value: data.EndedAt}
			default:
				log.From(ctx).Warn().Str("field", k).Msg("unknown field registered as cursor")
				delete(pc.Cursors, k)
			}
		}

		cursor, err := pc.Encode()
		if err != nil {
			log.From(ctx).Error().Err(err).Interface("page_cursor", pc).Msg("error encoding cursor")
		}
		var cron *string
		if data.CronSchedule.Valid {
			cron = &data.CronSchedule.String
		}
		var batchID *ulid.ULID
		isBatch := data.BatchID != nilULID
		if isBatch {
			batchID = &data.BatchID
		}

		res = append(res, &cqrs.TraceRun{
			AppID:        data.AppID,
			FunctionID:   data.FunctionID,
			TraceID:      string(data.TraceID),
			RunID:        data.RunID.String(),
			QueuedAt:     time.UnixMilli(data.QueuedAt),
			StartedAt:    time.UnixMilli(data.StartedAt),
			EndedAt:      time.UnixMilli(data.EndedAt),
			SourceID:     data.SourceID,
			TriggerIDs:   data.EventIDs(),
			Triggers:     [][]byte{},
			Output:       data.Output,
			Status:       enums.RunCodeToStatus(data.Status),
			IsBatch:      isBatch,
			BatchID:      batchID,
			IsDebounce:   data.IsDebounce,
			HasAI:        data.HasAi,
			CronSchedule: cron,
			Cursor:       cursor,
		})
		count++
		// enough items, don't need to proceed anymore
		if opt.Items > 0 && count >= opt.Items {
			break
		}
	}

	return res, nil
}

//
// Connect
//

func (w wrapper) InsertWorkerConnection(ctx context.Context, conn *cqrs.WorkerConnection) error {
	buildId := sql.NullString{}
	if conn.BuildId != nil {
		buildId.Valid = true
		buildId.String = *conn.BuildId
	}

	var lastHeartbeatAt, disconnectedAt sql.NullInt64

	if conn.LastHeartbeatAt != nil {
		lastHeartbeatAt = sql.NullInt64{
			Int64: conn.LastHeartbeatAt.UnixMilli(),
			Valid: true,
		}
	}

	if conn.DisconnectedAt != nil {
		lastHeartbeatAt = sql.NullInt64{
			Int64: conn.DisconnectedAt.UnixMilli(),
			Valid: true,
		}
	}

	var disconnectReason sql.NullString
	if conn.DisconnectReason != nil {
		disconnectReason = sql.NullString{
			String: *conn.DisconnectReason,
			Valid:  true,
		}
	}

	params := sqlc.InsertWorkerConnectionParams{
		AccountID:   conn.AccountID,
		WorkspaceID: conn.WorkspaceID,
		AppID:       conn.AppID,

		ID:         conn.Id,
		GatewayID:  conn.GatewayId,
		InstanceID: conn.InstanceId,
		Status:     int64(conn.Status),
		WorkerIp:   conn.WorkerIP,

		ConnectedAt:     conn.ConnectedAt.UnixMilli(),
		LastHeartbeatAt: lastHeartbeatAt,
		DisconnectedAt:  disconnectedAt,
		RecordedAt:      conn.RecordedAt.UnixMilli(),
		InsertedAt:      time.Now().UnixMilli(),

		DisconnectReason: disconnectReason,

		GroupHash:     []byte(conn.GroupHash),
		SdkLang:       conn.SDKLang,
		SdkVersion:    conn.SDKVersion,
		SdkPlatform:   conn.SDKPlatform,
		SyncID:        conn.SyncID,
		BuildID:       buildId,
		FunctionCount: int64(conn.FunctionCount),

		CpuCores: int64(conn.CpuCores),
		MemBytes: conn.MemBytes,
		Os:       conn.Os,
	}

	return w.q.InsertWorkerConnection(ctx, params)
}

type WorkerConnectionCursorFilter struct {
	ID    string
	Value int64
}

func (w wrapper) GetWorkerConnection(ctx context.Context, id cqrs.WorkerConnectionIdentifier) (*cqrs.WorkerConnection, error) {
	conn, err := w.q.GetWorkerConnection(ctx, sqlc.GetWorkerConnectionParams{
		AccountID:    id.AccountID,
		WorkspaceID:  id.WorkspaceID,
		ConnectionID: id.ConnectionID,
	})
	if err != nil {
		return nil, err
	}

	connectedAt := time.UnixMilli(conn.ConnectedAt)

	var disconnectedAt, lastHeartbeatAt *time.Time
	if conn.DisconnectedAt.Valid {
		disconnectedAt = ptr.Time(time.UnixMilli(conn.DisconnectedAt.Int64))
	}

	if conn.LastHeartbeatAt.Valid {
		lastHeartbeatAt = ptr.Time(time.UnixMilli(conn.LastHeartbeatAt.Int64))
	}

	var buildId *string
	if conn.BuildID.Valid {
		buildId = &conn.BuildID.String
	}

	var disconnectReason *string
	if conn.DisconnectReason.Valid {
		disconnectReason = &conn.DisconnectReason.String
	}

	workerConn := cqrs.WorkerConnection{
		AccountID:   conn.AccountID,
		WorkspaceID: conn.WorkspaceID,
		AppID:       conn.AppID,

		Id:         conn.ID,
		GatewayId:  conn.GatewayID,
		InstanceId: conn.InstanceID,
		Status:     connpb.ConnectionStatus(conn.Status),
		WorkerIP:   conn.WorkerIp,

		LastHeartbeatAt: lastHeartbeatAt,
		ConnectedAt:     connectedAt,
		DisconnectedAt:  disconnectedAt,
		RecordedAt:      time.UnixMilli(conn.RecordedAt),
		InsertedAt:      time.UnixMilli(conn.InsertedAt),

		DisconnectReason: disconnectReason,

		GroupHash:     string(conn.GroupHash),
		SDKLang:       conn.SdkLang,
		SDKVersion:    conn.SdkVersion,
		SDKPlatform:   conn.SdkPlatform,
		SyncID:        conn.SyncID,
		BuildId:       buildId,
		FunctionCount: int(conn.FunctionCount),

		CpuCores: int32(conn.CpuCores),
		MemBytes: conn.MemBytes,
		Os:       conn.Os,
	}

	return &workerConn, nil
}

type workerConnectionsQueryBuilder struct {
	filter       []sq.Expression
	order        []sqexp.OrderedExpression
	cursor       *cqrs.WorkerConnectionPageCursor
	cursorLayout *cqrs.WorkerConnectionPageCursor
}

func newWorkerConnectionsQueryBuilder(ctx context.Context, opt cqrs.GetWorkerConnectionOpt) *workerConnectionsQueryBuilder {
	// filters
	filter := []sq.Expression{}
	if len(opt.Filter.AppID) > 0 {
		filter = append(filter, sq.C("app_id").In(opt.Filter.AppID))
	}
	if len(opt.Filter.Status) > 0 {
		status := []int64{}
		for _, s := range opt.Filter.Status {
			status = append(status, int64(s))
		}
		filter = append(filter, sq.C("status").In(status))
	}
	tsfield := strings.ToLower(opt.Filter.TimeField.String())
	filter = append(filter, sq.C(tsfield).Gte(opt.Filter.From.UnixMilli()))

	until := opt.Filter.Until
	if until.UnixMilli() <= 0 {
		until = time.Now()
	}
	filter = append(filter, sq.C(tsfield).Lt(until.UnixMilli()))

	// Layout to be used for the response cursors
	resCursorLayout := cqrs.WorkerConnectionPageCursor{
		Cursors: map[string]cqrs.WorkerConnectionCursor{},
	}

	reqcursor := &cqrs.WorkerConnectionPageCursor{}
	if opt.Cursor != "" {
		if err := reqcursor.Decode(opt.Cursor); err != nil {
			log.From(ctx).Error().Err(err).Str("cursor", opt.Cursor).Msg("error decoding worker connection history cursor")
		}
	}

	// order by
	//
	// When going through the sorting fields, construct
	// - response pagination cursor layout
	// - update filter with op against sorted fields for pagination
	sortOrder := []enums.WorkerConnectionTimeField{}
	sortDir := map[enums.WorkerConnectionTimeField]enums.WorkerConnectionSortOrder{}
	cursorFilter := map[enums.WorkerConnectionTimeField]WorkerConnectionCursorFilter{}
	for _, f := range opt.Order {
		sortDir[f.Field] = f.Direction
		found := false
		for _, field := range sortOrder {
			if f.Field == field {
				found = true
				break
			}
		}
		if !found {
			sortOrder = append(sortOrder, f.Field)
		}

		rc := reqcursor.Find(f.Field.String())
		if rc != nil {
			cursorFilter[f.Field] = WorkerConnectionCursorFilter{ID: reqcursor.ID, Value: rc.Value}
		}
		resCursorLayout.Add(f.Field.String())
	}

	order := []sqexp.OrderedExpression{}
	for _, f := range sortOrder {
		var o sqexp.OrderedExpression
		field := strings.ToLower(f.String())
		if d, ok := sortDir[f]; ok {
			switch d {
			case enums.WorkerConnectionSortOrderAsc:
				o = sq.C(field).Asc()
			case enums.WorkerConnectionSortOrderDesc:
				o = sq.C(field).Desc()
			default:
				log.From(ctx).Error().Str("field", field).Str("direction", d.String()).Msg("invalid direction specified for sorting")
				continue
			}

			order = append(order, o)
		}
	}
	order = append(order, sq.C("id").Asc())

	// cursor filter
	for k, cf := range cursorFilter {
		ord, ok := sortDir[k]
		if !ok {
			continue
		}

		var compare sq.Expression
		field := strings.ToLower(k.String())
		switch ord {
		case enums.WorkerConnectionSortOrderAsc:
			compare = sq.C(field).Gt(cf.Value)
		case enums.WorkerConnectionSortOrderDesc:
			compare = sq.C(field).Lt(cf.Value)
		default:
			continue
		}

		filter = append(filter, sq.Or(
			compare,
			sq.And(
				sq.C(field).Eq(cf.Value),
				sq.C("id").Gt(cf.ID),
			),
		))
	}

	return &workerConnectionsQueryBuilder{
		filter:       filter,
		order:        order,
		cursor:       reqcursor,
		cursorLayout: &resCursorLayout,
	}
}

func (w wrapper) GetWorkerConnectionsCount(ctx context.Context, opt cqrs.GetWorkerConnectionOpt) (int, error) {
	// explicitly set it to zero so it would not attempt to paginate
	opt.Items = 0
	res, err := w.GetWorkerConnections(ctx, opt)
	if err != nil {
		return 0, err
	}

	return len(res), nil
}

func (w wrapper) GetWorkerConnections(ctx context.Context, opt cqrs.GetWorkerConnectionOpt) ([]*cqrs.WorkerConnection, error) {
	builder := newWorkerConnectionsQueryBuilder(ctx, opt)
	filter := builder.filter
	order := builder.order
	reqcursor := builder.cursor
	resCursorLayout := builder.cursorLayout

	// read from database
	// TODO:
	// change this to a continuous loop with limits instead of just attempting to grab everything.
	// might not matter though since this is primarily meant for local development
	sql, args, err := sq.Dialect(w.dialect()).
		From("worker_connections").
		Select(
			"account_id",
			"workspace_id",
			"app_id",

			"id",
			"gateway_id",
			"instance_id",
			"status",
			"worker_ip",

			"connected_at",
			"last_heartbeat_at",
			"disconnected_at",
			"recorded_at",
			"inserted_at",

			"disconnect_reason",

			"group_hash",
			"sdk_lang",
			"sdk_version",
			"sdk_platform",
			"sync_id",
			"build_id",
			"function_count",

			"cpu_cores",
			"mem_bytes",
			"os",
		).
		Where(filter...).
		Order(order...).
		ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := w.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	res := []*cqrs.WorkerConnection{}
	var count uint
	for rows.Next() {
		data := sqlc.WorkerConnection{}
		err := rows.Scan(
			&data.AccountID,
			&data.WorkspaceID,
			&data.AppID,

			&data.ID,
			&data.GatewayID,
			&data.InstanceID,
			&data.Status,
			&data.WorkerIp,

			&data.ConnectedAt,
			&data.LastHeartbeatAt,
			&data.DisconnectedAt,
			&data.RecordedAt,
			&data.InsertedAt,

			&data.DisconnectReason,

			&data.GroupHash,
			&data.SdkLang,
			&data.SdkVersion,
			&data.SdkPlatform,
			&data.SyncID,
			&data.BuildID,
			&data.FunctionCount,

			&data.CpuCores,
			&data.MemBytes,
			&data.Os,
		)
		if err != nil {
			return nil, err
		}

		// the cursor target should be skipped
		if reqcursor.ID == data.ID.String() {
			continue
		}

		// copy layout
		pc := resCursorLayout
		// construct the needed fields to generate a cursor representing this run
		pc.ID = data.ID.String()
		for k := range pc.Cursors {
			switch k {
			case strings.ToLower(enums.WorkerConnectionTimeFieldConnectedAt.String()):
				pc.Cursors[k] = cqrs.WorkerConnectionCursor{Field: k, Value: data.ConnectedAt}
			case strings.ToLower(enums.WorkerConnectionTimeFieldLastHeartbeatAt.String()):
				pc.Cursors[k] = cqrs.WorkerConnectionCursor{Field: k, Value: data.LastHeartbeatAt.Int64}
			case strings.ToLower(enums.WorkerConnectionTimeFieldDisconnectedAt.String()):
				pc.Cursors[k] = cqrs.WorkerConnectionCursor{Field: k, Value: data.DisconnectedAt.Int64}
			default:
				log.From(ctx).Warn().Str("field", k).Msg("unknown field registered as cursor")
				delete(pc.Cursors, k)
			}
		}

		cursor, err := pc.Encode()
		if err != nil {
			log.From(ctx).Error().Err(err).Interface("page_cursor", pc).Msg("error encoding cursor")
		}

		connectedAt := time.UnixMilli(data.ConnectedAt)

		var disconnectedAt, lastHeartbeatAt *time.Time
		if data.DisconnectedAt.Valid {
			disconnectedAt = ptr.Time(time.UnixMilli(data.DisconnectedAt.Int64))
		}
		if data.LastHeartbeatAt.Valid {
			lastHeartbeatAt = ptr.Time(time.UnixMilli(data.LastHeartbeatAt.Int64))
		}

		var buildId *string
		if data.BuildID.Valid {
			buildId = &data.BuildID.String
		}

		var disconnectReason *string
		if data.DisconnectReason.Valid {
			disconnectReason = &data.DisconnectReason.String
		}

		res = append(res, &cqrs.WorkerConnection{
			AccountID:   data.AccountID,
			WorkspaceID: data.WorkspaceID,
			AppID:       data.AppID,

			Id:         data.ID,
			GatewayId:  data.GatewayID,
			InstanceId: data.InstanceID,
			Status:     connpb.ConnectionStatus(data.Status),
			WorkerIP:   data.WorkerIp,

			LastHeartbeatAt: lastHeartbeatAt,
			ConnectedAt:     connectedAt,
			DisconnectedAt:  disconnectedAt,
			RecordedAt:      time.UnixMilli(data.RecordedAt),
			InsertedAt:      time.UnixMilli(data.InsertedAt),

			DisconnectReason: disconnectReason,

			GroupHash:     string(data.GroupHash),
			SDKLang:       data.SdkLang,
			SDKVersion:    data.SdkVersion,
			SDKPlatform:   data.SdkPlatform,
			SyncID:        data.SyncID,
			FunctionCount: int(data.FunctionCount),
			BuildId:       buildId,

			CpuCores: int32(data.CpuCores),
			MemBytes: data.MemBytes,
			Os:       data.Os,

			Cursor: cursor,
		})
		count++
		// enough items, don't need to proceed anymore
		if opt.Items > 0 && count >= opt.Items {
			break
		}
	}

	return res, nil
}

// copyWriter allows running duck-db specific functions as CQRS functions, copying CQRS types to DDB types
// automatically.
func copyWriter[
	PARAMS_IN any,
	INTERNAL_PARAMS any,
	IN any,
	OUT any,
](
	ctx context.Context,
	f func(context.Context, INTERNAL_PARAMS) (IN, error),
	pin PARAMS_IN,
	pout INTERNAL_PARAMS,
	out OUT,
) (OUT, error) {
	err := copier.Copy(&pout, &pin)
	if err != nil {
		return out, err
	}

	in, err := f(ctx, pout)
	if err != nil {
		return out, err
	}

	err = copier.CopyWithOption(&out, in, copier.Option{DeepCopy: true})
	return out, err
}

func copyInto[
	IN any,
	OUT any,
](
	ctx context.Context,
	f func(context.Context) (IN, error),
	out OUT,
) (OUT, error) {
	in, err := f(ctx)
	if err != nil {
		return out, err
	}
	err = copier.CopyWithOption(&out, in, copier.Option{DeepCopy: true})
	return out, err
}
