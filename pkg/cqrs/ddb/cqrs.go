package ddb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/ddb/sqlc"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jinzhu/copier"
	"github.com/oklog/ulid/v2"
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
	fn, err := w.GetFunctionByID(ctx, identifier.WorkflowID)
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

func (w wrapper) GetAppByURL(ctx context.Context, url string) (*cqrs.App, error) {
	// Normalize the URL before inserting into the DB.
	url = util.NormalizeAppURL(url)

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
	arg.Url = util.NormalizeAppURL(arg.Url)

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
	arg.Url = util.NormalizeAppURL(arg.Url)

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

func (w wrapper) GetFunctionByID(ctx context.Context, id uuid.UUID) (*cqrs.Function, error) {
	f := func(ctx context.Context) (*sqlc.Function, error) {
		return w.q.GetFunctionByID(ctx, id)
	}
	return copyInto(ctx, f, &cqrs.Function{})
}

func (w wrapper) GetFunctions(ctx context.Context) ([]*cqrs.Function, error) {
	return copyInto(ctx, w.q.GetFunctions, []*cqrs.Function{})
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

func (w wrapper) GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*cqrs.Event, error) {
	obj, err := w.q.GetEventByInternalID(ctx, internalID)
	if err != nil {
		return nil, fmt.Errorf("error quering event in ddb: %w", err)
	}
	evt := convertEvent(obj)
	return &evt, nil
}
func (w wrapper) GetEventsTimebound(ctx context.Context, t cqrs.Timebound, limit int) ([]*cqrs.Event, error) {
	after := time.Time{}                           // after the beginning of time, eg all
	before := time.Now().Add(time.Hour * 24 * 365) // before 1 year in the future, eg all
	if t.After != nil {
		after = *t.After
	}
	if t.Before != nil {
		before = *t.Before
	}

	evts, err := w.q.GetEventsTimebound(ctx, sqlc.GetEventsTimeboundParams{
		After:  after,
		Before: before,
		Limit:  int64(limit),
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

//
// Function runs
//

func (w wrapper) InsertFunctionRun(ctx context.Context, e cqrs.FunctionRun) error {
	run := sqlc.InsertFunctionRunParams{}
	if err := copier.CopyWithOption(&run, e, copier.Option{DeepCopy: true}); err != nil {
		return err
	}
	return w.q.InsertFunctionRun(ctx, run)
}

func (w wrapper) GetFunctionRunsFromEvents(ctx context.Context, eventIDs []ulid.ULID) ([]*cqrs.FunctionRun, error) {
	return copyInto(ctx, func(ctx context.Context) ([]*sqlc.FunctionRun, error) {
		return w.q.GetFunctionRunsFromEvents(ctx, eventIDs)
	}, []*cqrs.FunctionRun{})
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

	return copyInto(ctx, func(ctx context.Context) ([]*sqlc.FunctionRun, error) {
		return w.q.GetFunctionRunsTimebound(ctx, sqlc.GetFunctionRunsTimeboundParams{
			Before: before,
			After:  after,
			Limit:  int64(limit),
		})
	}, []*cqrs.FunctionRun{})
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
