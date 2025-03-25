package util

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
)

// ConcurrencyKey generates a concurrency key scoped appropriately.
func ConcurrencyKey(scope enums.ConcurrencyScope, scopeID uuid.UUID, unhashed string) string {
	return fmt.Sprintf("%s:%s:%s", ConcurrencyScopePrefix(scope), scopeID, XXHash(unhashed))

}

func ConcurrencyScopePrefix(scope enums.ConcurrencyScope) string {
	switch scope {
	case enums.ConcurrencyScopeFn:
		return "f"
	case enums.ConcurrencyScopeEnv:
		return "e"
	case enums.ConcurrencyScopeAccount:
		return "a"
	default:
		return "?"
	}
}
