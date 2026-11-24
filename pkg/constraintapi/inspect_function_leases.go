package constraintapi

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/rueidis"
)

//go:embed lua/inspect_function_leases.lua
var inspectFunctionLeasesSource string

var inspectFunctionLeasesScript = rueidis.NewLuaScript(inspectFunctionLeasesSource)

// CheckFunctionLeasesDrained checks ALL indexed account leases, including expired
// leases that a scavenger could still release. Unknown details fail closed.
// This is maintenance-only and is not part of Acquire, Release or scavenging.
// It is read-only: an empty index does not rule out unindexed holders and must
// never be used as authorization to delete semaphore counters.
func CheckFunctionLeasesDrained(ctx context.Context, client rueidis.Client, accountID, functionID uuid.UUID) error {
	if client == nil || accountID == uuid.Nil || functionID == uuid.Nil {
		return fmt.Errorf("retirement requires client and complete scope")
	}
	prefix := fmt.Sprintf("{cs}:%s", accountScope(accountID))
	_, err := inspectFunctionLeasesScript.Exec(ctx, client, []string{prefix + ":leaseq"}, []string{prefix, functionID.String(), "1000"}).AsInt64()
	if err != nil {
		return fmt.Errorf("function retirement refused: %w", err)
	}
	return nil
}
