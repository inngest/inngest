package base_cqrs

import (
	"github.com/inngest/inngest/pkg/cqrs/manager"
	"github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/history_reader"
)

// NewHistoryReader returns the legacy history reader implementation.
//
// Deprecated: history read access is moving to pkg/cqrs/manager.
func NewHistoryReader(adapter db.Adapter) history_reader.Reader {
	return manager.NewHistoryReader(adapter)
}
