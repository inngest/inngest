package base_cqrs

import (
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
)

func SQLiteToCQRSFunction(fn sqlc.Function) cqrs.Function {
	var archivedAt time.Time
	if fn.ArchivedAt.Valid {
		archivedAt = fn.ArchivedAt.Time
	}

	return cqrs.Function{
		ID:         fn.ID,
		AppID:      fn.AppID,
		Name:       fn.Name,
		Slug:       fn.Slug,
		Config:     json.RawMessage(fn.Config),
		CreatedAt:  fn.CreatedAt,
		ArchivedAt: archivedAt,
	}
}
