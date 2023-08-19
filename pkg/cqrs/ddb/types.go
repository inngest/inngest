package ddb

import (
	"github.com/inngest/inngest/pkg/cqrs/ddb/sqlc"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/jinzhu/copier"
)

func convertHistoryToWriter(h history.History) (*sqlc.InsertHistoryParams, error) {
	to := sqlc.InsertHistoryParams{}
	if err := copier.CopyWithOption(&to, h, copier.Option{DeepCopy: true}); err != nil {
		return nil, err
	}

	return &to, nil
}
