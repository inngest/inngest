package base_cqrs

import (
	dbpkg "github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/jinzhu/copier"
)

func convertHistoryToWriter(h history.History) (*dbpkg.InsertHistoryParams, error) {
	to := dbpkg.InsertHistoryParams{}
	if err := copier.CopyWithOption(&to, h, copier.Option{DeepCopy: true}); err != nil {
		return nil, err
	}

	return &to, nil
}
