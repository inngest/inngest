package driver

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/inngest/inngest/pkg/duckdb/worker"
)

// workerSessionExecer adapts a *worker.Session into the driver's sqlExecer and
// streamQuerier interfaces, enforcing WithMaxResultBytes and normalizing
// DuckDB values into database/sql-compatible Go types.
type workerSessionExecer struct {
	sess *worker.Session
}

func (w *workerSessionExecer) Exec(ctx context.Context, sqlText string) ([]string, []result.Row, error) {
	cols, _, rows, err := w.Query(ctx, sqlText)
	return cols, rows, err
}

func (w *workerSessionExecer) Query(ctx context.Context, sqlText string) ([]string, []string, []result.Row, error) {
	qs, err := w.sess.Query(ctx, sqlText)
	if err != nil {
		return nil, nil, nil, translateWorkerErr(err)
	}
	return drainQueryStream(ctx, w.sess, qs)
}

func (w *workerSessionExecer) QueryWithStreams(ctx context.Context, sqlText string, streams []worker.SendStream) ([]string, []string, []result.Row, error) {
	qs, err := w.sess.QueryWithStreams(ctx, sqlText, streams)
	if err != nil {
		return nil, nil, nil, translateWorkerErr(err)
	}
	return drainQueryStream(ctx, w.sess, qs)
}

func drainQueryStream(ctx context.Context, sess *worker.Session, qs *worker.QueryStream) ([]string, []string, []result.Row, error) {
	defer qs.Close()

	maxBytes := result.MaxBytes(ctx)
	if maxBytes > 0 && qs.BytesRead() > maxBytes {
		interruptAsync(sess)
		return nil, nil, nil, result.TooLargeError(maxBytes)
	}

	wireCols := qs.Columns()
	colNames := make([]string, len(wireCols))
	colTypes := make([]string, len(wireCols))
	for i, c := range wireCols {
		colNames[i] = c.Name
		colTypes[i] = c.TypeName
	}
	rc := result.NewColumns(colNames)

	var rows []result.Row
	for {
		chunk, err := qs.NextChunk()
		if err != nil {
			return nil, nil, nil, translateWorkerErr(err)
		}
		if maxBytes > 0 && qs.BytesRead() > maxBytes {
			interruptAsync(sess)
			return nil, nil, nil, result.TooLargeError(maxBytes)
		}
		if chunk == nil {
			break
		}
		for _, r := range chunk {
			vals := make([]any, len(r))
			for i, v := range r {
				vals[i] = normalizeDriverValue(v)
			}
			rows = append(rows, result.Row{Cols: rc, Vals: vals})
		}
	}
	return colNames, colTypes, rows, nil
}

func interruptAsync(sess *worker.Session) {
	intCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = sess.Interrupt(intCtx)
}

func translateWorkerErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, worker.ErrStreamSource) {
		return err
	}
	if errors.Is(err, worker.ErrStatementFailed) {
		return fmt.Errorf("%w: %w", result.ErrStatementFailed, err)
	}
	return err
}

// normalizeDriverValue widens fixed-width integer/float types into the
// database/sql/driver.Value-compatible Go types that driver callers and tests
// expect (matching the previous quack transport's Go value contract).
func normalizeDriverValue(v any) any {
	switch x := v.(type) {
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case uint8:
		return int64(x)
	case uint16:
		return int64(x)
	case uint32:
		return int64(x)
	case uint64:
		return strconv.FormatUint(x, 10)
	case float32:
		return float64(x)
	default:
		return v
	}
}
