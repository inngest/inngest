package debug

import (
	debugpkg "github.com/inngest/inngest/pkg/debug"
)

// Re-export for backward compatibility
var DbgCtxKey = debugpkg.CtxKey
type DebugContext = debugpkg.Context
