**Problem statement**

- Inngest executes Steps in the SDK.
- Steps can error, then retry.  When steps reach the total number of retries and still fail, the step *fails.*
    - Users can try/catch this final step failure to handle permanent step errors gracefully: `try { await step.run("", () => {}) } catch(err) {}`
    - Users can use multiple return values in Go to do the same thing.
- We currently use Opcodes to determine what a step does: `OpcodeStepRun`, `OpcodeStepSleep`, `OpcodeStepError`.
- Unfortunately for us, **permanent step failures use the same opcode as temporary, retryable step errors, making our code and logic messy.**
- We should **implement a new `OpcodeStepFailed` enum to the executor to simplify tracing, SDK logic, checkpointing, and so on.**

**User stories**

- This is purely backend facing, for now.

**Documentation**

- None

**Pricing/Packaging**

- This does not impact pricing, packaging, or tiers

**Out of scope**

- We are not exposing this to users.
- We are not changing the UI or any trace / error displays

**Goals, Success & failure metrics**

- Code starts to use OpcodeStepFailed and is simpler.

**Related features**

- [Per-step error handling](https://www.notion.so/Per-step-error-handling-e33d9988446345b5a8b2e299a0fdffa0?pvs=21)
- [Improve errors for users](https://www.notion.so/Improve-errors-for-users-693e47d9f86b4f42ad5b0ac5a302a7cb?pvs=21)

# Implementation

## Overview

This implementation adds semantic distinction between retryable step errors and permanent step failures by introducing a new `OpcodeStepFailed` opcode. Currently, both scenarios use `OpcodeStepError`, making it impossible to distinguish between temporary failures (should retry) and permanent failures (exhausted retries or non-retryable errors).

## Phase 1: Core Infrastructure Changes

### 1. Opcode Enum Definition (`pkg/enums/opcode.go`)

**Current state:** `OpcodeStepError = 3` is used for both retryable and permanent failures

**Changes needed:**
```go
const (
    OpcodeNone        Opcode = iota
    OpcodeStep               
    OpcodeStepRun            
    OpcodeStepError          // Keep for retryable errors and backward compatibility
    OpcodeStepFailed         // NEW: Add after OpcodeStepError  
    OpcodeStepPlanned        // This will shift from 4 to 5
    OpcodeSleep              // This will shift from 5 to 6
    // ... rest shift by 1
)
```

**Critical:** Must add to `opcodeSyncMap` since step failures are synchronous:
```go
var opcodeSyncMap = map[Opcode]struct{}{
    OpcodeStep:        {},
    OpcodeStepRun:     {},
    OpcodeStepFailed:  {}, // NEW: Add this line
    OpcodeRunComplete: {},
}
```

**After changes:** Run `go generate` to regenerate `opcode_enumer.go`

### 2. Executor Logic (`pkg/execution/executor/executor.go`)

**Current flow:** All step errors go to `handleStepError()` which determines retryability internally

**Changes needed:**

#### A. Update main handler switch (line ~2346):
```go
switch gen.Op {
case enums.OpcodeStepError:
    return e.handleStepError(ctx, runCtx, gen, edge)
case enums.OpcodeStepFailed:  // NEW: Add this case
    return e.handleStepFailed(ctx, runCtx, gen, edge)
// ... rest of cases
```

#### B. Extract permanent failure logic from `handleStepError()`:
Move lines 2536-2590 (the "This was the final step attempt" section) to new `handleStepFailed()` function.

#### C. Modify `handleStepError()` logic (line ~2514):
```go
retryable := true

if gen.Error.NoRetry {
    // This is a NonRetryableError - should be OpcodeStepFailed
    // but older SDKs send OpcodeStepError, so convert internally
    retryable = false
}
if !runCtx.ShouldRetry() {
    // Max retries reached - should be OpcodeStepFailed  
    // but older SDKs send OpcodeStepError, so convert internally
    retryable = false
}

if retryable {
    // Still retryable - proceed with retry logic
    runCtx.IncrementAttempt()
    // ... existing retry logic
    return ErrHandledStepError
}

// For backward compatibility: convert OpcodeStepError to OpcodeStepFailed
// when it represents a permanent failure from older SDKs
gen.Op = enums.OpcodeStepFailed
return e.handleStepFailed(ctx, runCtx, gen, edge)
```

### 3. Tracing Support (`pkg/tracing/util.go`)

**Current:** `OpcodeStepError` handling at line 202-221

**Changes needed:**
Add `OpcodeStepFailed` to the switch statement (line ~157):
```go
case enums.OpcodeStep, enums.OpcodeStepRun, enums.OpcodeStepError, enums.OpcodeStepFailed:
    {
        // Output (success or error)
        if output, err := op.Output(); err == nil {
            meta.AddAttr(rawAttrs, meta.Attrs.StepOutput, &output)
        } else {
            rawAttrs.AddErr(fmt.Errorf("failed to get step output: %w", err))
        }

        // Set status if we've encountered an error
        if op.Error != nil {
            status := enums.StepStatusErrored
            meta.AddAttr(rawAttrs, meta.Attrs.DynamicStatus, &status)
        }
    }
```

## Phase 2: SDK Integration

### Current SDK Behavior Analysis

The executor currently has the logic to distinguish between retryable and non-retryable errors:
- `gen.Error.NoRetry = true` → NonRetryableError from SDK  
- `!runCtx.ShouldRetry()` → Max retries reached

**For backward compatibility:** Initially, keep SDKs sending `OpcodeStepError` and let the executor convert to `OpcodeStepFailed` internally.

**Future enhancement:** Update SDKs to send `OpcodeStepFailed` directly for permanent failures.

## Phase 3: Testing Strategy

### 1. Unit Tests

#### A. Opcode Enum Tests
File: `pkg/enums/opcode_test.go` (create if doesn't exist)
```go
func TestOpcodeStepFailedIsSync(t *testing.T) {
    require.True(t, OpcodeIsSync(enums.OpcodeStepFailed))
}

func TestOpcodeStepFailedString(t *testing.T) {
    require.Equal(t, "StepFailed", enums.OpcodeStepFailed.String())
}
```

#### B. Executor Logic Tests  
File: `pkg/execution/executor/executor_test.go`
- Test `handleStepFailed()` function
- Test conversion from `OpcodeStepError` to `OpcodeStepFailed` in `handleStepError()`
- Test backward compatibility with older SDK responses

### 2. Integration Tests - Go SDK

#### A. Extend Existing Tests
File: `tests/golang/step_error_test.go`

Add new test function:
```go
func TestStepFailedOpcode(t *testing.T) {
    // Test that after max retries, we can distinguish the final failure
}

func TestNonRetryableStepFailed(t *testing.T) {
    // Test NonRetryableError immediately generates appropriate opcode
}
```

#### B. New Integration Test File
File: `tests/golang/opcode_verification_test.go`
```go
func TestOpcodeStepFailedTracing(t *testing.T) {
    // Create function with retryable step that fails
    // Verify SQLite database shows:
    // - OpcodeStepError for retries 1-2  
    // - OpcodeStepFailed for final failure
    
    // Query: SELECT json_extract(attributes, '$."_inngest.step.op"') 
    // FROM spans WHERE run_id = ? ORDER BY start_time
}
```

### 3. Integration Tests - TypeScript SDK

#### A. Basic Test
File: `tests/js/src/inngest/step_failed_test.ts`
```typescript
export const testStepFailed = inngest.createFunction(
  { id: "step-failed-test", retries: 2 },
  { event: "tests/step.failed" },
  async ({ step }) => {
    try {
      await step.run("retryable-failure", async () => {
        throw new Error("Always fails");
      });
    } catch (err) {
      // This should be a permanent failure after 3 attempts
      return { failedAsExpected: true };
    }
  }
);

export const testNonRetryableFailed = inngest.createFunction(
  { id: "non-retryable-test" },
  { event: "tests/non-retryable" }, 
  async ({ step }) => {
    try {
      await step.run("immediate-failure", async () => {
        throw new NonRetriableError("Bad config");
      });
    } catch (err) {
      return { failedImmediately: true };
    }
  }
);
```
```
```

## Verification & Debugging

### SQLite Queries for Testing

During development, use these queries to verify correct opcode generation:

```sql
-- See opcode progression for a specific run
SELECT 
  json_extract(attributes, '$."_inngest.step.name"') as step_name,
  json_extract(attributes, '$."_inngest.step.op"') as opcode,
  json_extract(attributes, '$."_inngest.dynamic.status"') as status,
  datetime(start_time) as timestamp
FROM spans 
WHERE run_id = 'your-run-id'
  AND json_extract(attributes, '$."_inngest.step.op"') IS NOT NULL
ORDER BY start_time;

-- Count opcode distribution
SELECT 
  json_extract(attributes, '$."_inngest.step.op"') as opcode,
  COUNT(*) as count
FROM spans 
WHERE json_extract(attributes, '$."_inngest.step.op"') IS NOT NULL
GROUP BY json_extract(attributes, '$."_inngest.step.op"');

-- Find runs with both StepError and StepFailed
SELECT 
  run_id,
  GROUP_CONCAT(DISTINCT json_extract(attributes, '$."_inngest.step.op"')) as opcodes
FROM spans 
WHERE json_extract(attributes, '$."_inngest.step.op"') IN ('StepError', 'StepFailed')
GROUP BY run_id
HAVING opcodes LIKE '%StepError%' AND opcodes LIKE '%StepFailed%';
```

## Success Criteria

1. **Enum Integration:** `OpcodeStepFailed` appears in generated code and is recognized as sync
2. **Executor Logic:** Permanent failures (both max retries and NonRetryableError) generate `OpcodeStepFailed`
3. **Tracing:** SQLite database shows correct opcode progression: `StepError` → `StepError` → `StepFailed`
4. **Backward Compatibility:** Older SDK versions continue working without changes
5. **Integration Tests:** All test suites pass with both Go and TypeScript SDK tests

**Critical:** Never remove support for `OpcodeStepError` as older SDK versions will always exist in the wild.
