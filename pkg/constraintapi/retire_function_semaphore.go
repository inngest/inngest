package constraintapi

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/rueidis"
)

//go:embed lua/retire_function_semaphore.lua
var retireFunctionSemaphoreSource string

var retireFunctionSemaphoreScript = rueidis.NewLuaScript(retireFunctionSemaphoreSource)

// CheckFunctionLeasesDrained checks ALL indexed account leases, including expired
// leases that a scavenger could still release. Unknown details fail closed.
// This is maintenance-only and is not part of Acquire, Release or scavenging.
func CheckFunctionLeasesDrained(ctx context.Context, client rueidis.Client, accountID, functionID uuid.UUID) error {
	return checkOrRetireFunctionSemaphore(ctx, client, accountID, functionID, "", "", false)
}

// RetireFunctionSemaphoreBucket deletes an obsolete keyed function-concurrency
// counter, NOT an acquired permit. The caller must hold and quiesce the function,
// drain its entire old-format backlog, and ensure its current configuration no
// longer uses the expression. Never reactivate that namespace or roll back to
// old writers. A Lua check prevents deletion while any indexed lease for the
// function can still be released, even when the counter is zero or missing.
func RetireFunctionSemaphoreBucket(ctx context.Context, client rueidis.Client, accountID, functionID uuid.UUID, expression, bucket string) error {
	if expression == "" || bucket == "" {
		return fmt.Errorf("retirement requires a keyed function semaphore")
	}
	return checkOrRetireFunctionSemaphore(ctx, client, accountID, functionID, expression, bucket, true)
}

func checkOrRetireFunctionSemaphore(ctx context.Context, client rueidis.Client, accountID, functionID uuid.UUID, expression, bucket string, remove bool) error {
	if client == nil || accountID == uuid.Nil || functionID == uuid.Nil {
		return fmt.Errorf("retirement requires client and complete scope")
	}
	prefix := fmt.Sprintf("{cs}:%s", accountScope(accountID))
	usageKey := semaphoreUsageKey(accountID, SemaphoreIDFnKey(functionID, expression), bucket)
	mode := "check"
	if remove {
		mode = "retire"
	}
	_, err := retireFunctionSemaphoreScript.Exec(ctx, client, []string{prefix + ":leaseq", usageKey}, []string{prefix, functionID.String(), mode, "1000"}).AsInt64()
	if err != nil {
		return fmt.Errorf("function retirement refused: %w", err)
	}
	return nil
}
