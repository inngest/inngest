package base_cqrs

import (
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
)

func SQLiteToCQRSFunction(fn *sqlc.Function) *cqrs.Function {
	if fn == nil {
		return nil
	}

	var archivedAt time.Time
	if fn.ArchivedAt.Valid {
		archivedAt = fn.ArchivedAt.Time
	}

	return &cqrs.Function{
		ID:         fn.ID,
		AppID:      fn.AppID,
		Name:       fn.Name,
		Slug:       fn.Slug,
		Config:     json.RawMessage(fn.Config),
		CreatedAt:  fn.CreatedAt,
		ArchivedAt: archivedAt,
	}
}

func SQLiteToCQRSFunctionList(fns []*sqlc.Function) []*cqrs.Function {
	if len(fns) == 0 {
		return []*cqrs.Function{}
	}

	res := make([]*cqrs.Function, len(fns))
	for i, fn := range fns {
		res[i] = SQLiteToCQRSFunction(fn)
	}
	return res
}
