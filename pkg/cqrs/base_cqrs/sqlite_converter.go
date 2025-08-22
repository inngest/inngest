package base_cqrs

import (
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
)

// SQLiteToCQRS accepts an input and converter, and convert the type to another
func SQLiteToCQRS[T, R any](input *T, converter func(*T) *R) *R {
	if input == nil {
		return nil
	}
	return converter(input)
}

func SQLiteToCQRSList[T, R any](inputs []*T, converter func(*T) *R) []*R {
	if len(inputs) == 0 {
		return []*R{}
	}

	results := make([]*R, len(inputs))
	for i, input := range inputs {
		results[i] = SQLiteToCQRS(input, converter)
	}
	return results
}

//
// Converters
//

// convertSQLiteFunctionToCQRS converts sqlc function to cqrs function
func convertSQLiteFunctionToCQRS(fn *sqlc.Function) *cqrs.Function {
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
