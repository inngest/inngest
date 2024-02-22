package sqlitecqrs

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/sqlitecqrs/sqlc"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
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
)

func NewCQRS(db *sql.DB) cqrs.Manager {
	return wrapper{
		q:  sqlc.New(db),
		db: db,
	}
}

type wrapper struct {
	q  *sqlc.Queries
	db *sql.DB
	tx *sql.Tx
}

// LoadFunction implements the state.FunctionLoader interface.
func (w wrapper) LoadFunction(ctx context.Context, identifier state.Identifier) (*inngest.Function, error) {
	// XXX: This doesn't store versions, as the dev server is currently ignorant to version.s
	fn, err := w.GetFunctionByInternalUUID(ctx, identifier.WorkspaceID, identifier.WorkflowID)
	if err != nil {
		return nil, err
	}
	return fn.InngestFunction()
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
	return &wrapper{
		q:  sqlc.New(tx),
		tx: tx,
	}, nil
}

func (w wrapper) Commit(ctx context.Context) error {
	return w.tx.Commit()
}

func (w wrapper) Rollback(ctx context.Context) error {
	return w.tx.Rollback()
}

//
// Apps
//

// GetApps returns apps that have not been deleted.
func (w wrapper) GetApps(ctx context.Context) ([]*cqrs.App, error) {
	return copyInto(ctx, w.q.GetApps, []*cqrs.App{})
}

func (w wrapper) GetAppByChecksum(ctx context.Context, checksum string) (*cqrs.App, error) {
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

func (w wrapper) GetAppByURL(ctx context.Context, url string) (*cqrs.App, error) {
	// Normalize the URL before inserting into the DB.
	url = util.NormalizeAppURL(url, forceHTTPS)

	f := func(ctx context.Context) (*sqlc.App, error) {
		return w.q.GetAppByURL(ctx, url)
	}
	return copyInto(ctx, f, &cqrs.App{})
}

// GetAllApps returns all apps.
func (w wrapper) GetAllApps(ctx context.Context) ([]*cqrs.App, error) {
	return copyInto(ctx, w.q.GetAllApps, []*cqrs.App{})
}

// InsertApp creates a new app.
func (w wrapper) InsertApp(ctx context.Context, arg cqrs.InsertAppParams) (*cqrs.App, error) {
	// Normalize the URL before inserting into the DB.
	arg.Url = util.NormalizeAppURL(arg.Url, forceHTTPS)

	return copyWriter(
		ctx,
		w.q.InsertApp,
		arg,
		sqlc.InsertAppParams{},
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
	if err := w.q.HardDeleteApp(ctx, arg.ID); err != nil {
		return nil, err
	}

	app.Error = arg.Error
	params := sqlc.InsertAppParams{}
	_ = copier.CopyWithOption(&params, app, copier.Option{DeepCopy: true})

	// Recreate the app.
	app, err = w.q.InsertApp(ctx, params)
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
	if err := w.q.HardDeleteApp(ctx, arg.ID); err != nil {
		return nil, err
	}
	app.Url = arg.Url
	params := sqlc.InsertAppParams{}
	_ = copier.CopyWithOption(&params, app, copier.Option{DeepCopy: true})
	// Recreate the app.
	app, err = w.q.InsertApp(ctx, params)
	if err != nil {
		return nil, err
	}
	out := &cqrs.App{}
	err = copier.CopyWithOption(out, app, copier.Option{DeepCopy: true})
	return out, err
}

// DeleteApp deletes an app
func (w wrapper) DeleteApp(ctx context.Context, id uuid.UUID) error {
	return w.q.HardDeleteApp(ctx, id)
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

func (w wrapper) GetEventsTimebound(
	ctx context.Context,
	t cqrs.Timebound,
	limit int,
	includeInternal bool,
) ([]*cqrs.Event, error) {
	after := time.Time{}                           // after the beginning of time, eg all
	before := time.Now().Add(time.Hour * 24 * 365) // before 1 year in the future, eg all
	if t.After != nil {
		after = *t.After
	}
	if t.Before != nil {
		before = *t.Before
	}

	evts, err := w.q.GetEventsTimebound(ctx, sqlc.GetEventsTimeboundParams{
		After:           after,
		Before:          before,
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
	}
	if run.BatchID != nilULID {
		copied.BatchID = &run.BatchID
	}
	if run.OriginalRunID != nilULID {
		copied.BatchID = &run.OriginalRunID
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
