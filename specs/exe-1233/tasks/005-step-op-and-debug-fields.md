# Task exe-1233-005: Add Step Operation Type and Debug Run ID to StepInfo

**Feature**: exe-1233
**Status**: pending
**Priority**: P2
**Depends on**: none

## Objective

Add two fields to the V4 `StepInfo` expanded details section: the step operation type label (T1-11) with human-friendly mapping, and the debug run ID (T1-10) visible only in debug mode. Both fields are already on the `Trace` type but not rendered.

## Acceptance Criteria

- [ ] Tests written first (RED phase)
- [ ] Tests verified to fail before implementation
- [ ] Implementation passes all tests (GREEN phase)
- [ ] `stepOp` renders as human-friendly label (e.g., "INVOKE" → "Invoke")
- [ ] Unknown `stepOp` values render raw enum string as fallback
- [ ] `stepOp` field hidden when null/undefined
- [ ] `debugRunID` renders as monospace ID when `debug === true` and non-null
- [ ] `debugRunID` hidden when `debug === false` or value is null
- [ ] `STEP_OP_LABELS` mapping covers all known enum values
- [ ] V3 components are NOT modified
- [ ] Code follows project conventions

## TDD Requirements

**Test File**: `ui/packages/components/src/RunDetailsV4/StepInfo.test.tsx` (add to existing from tasks 001-003)
**Test Command**: `cd ui/packages/components && pnpm test -- --run StepInfo`

## Project Tooling

**Lint**: `cd ui/packages/components && pnpm lint`
**Format**: `cd ui && pnpm run format`
**Build**: `cd ui/apps/dashboard && pnpm build`

### Tests to Write

**Step Operation Type (T1-11):**
1. **Step op renders human-friendly label**: Render `StepInfo` with `trace.stepOp = "INVOKE"`. Assert "Step Type" label with text "Invoke".
2. **Step op maps all known values**: Test each mapping:
   - `"RUN"` → "Step Run"
   - `"INVOKE"` → "Invoke"
   - `"SLEEP"` → "Sleep"
   - `"WAIT_FOR_EVENT"` → "Wait for Event"
   - `"AI_GATEWAY"` → "AI Gateway"
   - `"WAIT_FOR_SIGNAL"` → "Wait for Signal"
3. **Unknown step op renders raw value**: Render with `trace.stepOp = "FUTURE_OP"`. Assert text "FUTURE_OP".
4. **Step op hidden when null**: Render with `trace.stepOp = null`. Assert no "Step Type" label in document.

**Debug Run ID (T1-10):**
5. **Debug Run ID renders when debug=true and non-null**: Render `StepInfo` with `debug = true`, `trace.debugRunID = "01DEBUG..."`. Assert "Debug Run ID" label and ID text present.
6. **Debug Run ID hidden when debug=false**: Render with `debug = false`, `trace.debugRunID = "01DEBUG..."`. Assert no "Debug Run ID" label.
7. **Debug Run ID hidden when null**: Render with `debug = true`, `trace.debugRunID = null`. Assert no "Debug Run ID" label.

**Testing notes**:
- Both fields are in the expanded details section of `StepInfo` (the `{expanded && (...)}` block).
- The component starts with `expanded = true` so fields are visible by default.
- Mock hooks same as task 001.

## Implementation Notes

**File to modify**: `ui/packages/components/src/RunDetailsV4/StepInfo.tsx`

### Step Op Label Mapping

Add this constant near the top of the file (after imports, before components):

```tsx
const STEP_OP_LABELS: Record<string, string> = {
  RUN: 'Step Run',
  INVOKE: 'Invoke',
  SLEEP: 'Sleep',
  WAIT_FOR_EVENT: 'Wait for Event',
  AI_GATEWAY: 'AI Gateway',
  WAIT_FOR_SIGNAL: 'Wait for Signal',
};
```

### Current Expanded Section (lines 221-258)

```tsx
{expanded && (
  <div className="flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4 px-4">
    {!trace.isUserland && (
      <ElementWrapper label="Queued at">
        <TimeElement date={new Date(trace.queuedAt)} />
      </ElementWrapper>
    )}

    <ElementWrapper label="Started at">
      {trace.startedAt ? (
        <TimeElement date={new Date(trace.startedAt)} />
      ) : (
        <TextElement>-</TextElement>
      )}
    </ElementWrapper>

    <ElementWrapper label="Ended at">
      {trace.endedAt ? (
        <TimeElement date={new Date(trace.endedAt)} />
      ) : (
        <TextElement>-</TextElement>
      )}
    </ElementWrapper>

    {!trace.isUserland && (
      <ElementWrapper label="Delay">
        <TextElement>{delayText}</TextElement>
      </ElementWrapper>
    )}

    <ElementWrapper label="Duration">
      <TextElement>{durationText}</TextElement>
    </ElementWrapper>

    {stepKindInfo}

    {aiOutput && <AITrace aiOutput={aiOutput} />}
  </div>
)}
```

### What to Add

Add step op AFTER the Duration element and BEFORE `{stepKindInfo}`:

```tsx
{/* T1-11: Step Operation Type */}
{trace.stepOp && (
  <ElementWrapper label="Step Type">
    <TextElement>{STEP_OP_LABELS[trace.stepOp] ?? trace.stepOp}</TextElement>
  </ElementWrapper>
)}
```

Add debug run ID AFTER `{stepKindInfo}` (conditional on `debug` prop):

```tsx
{/* T1-10: Debug Run ID */}
{debug && trace.debugRunID && (
  <ElementWrapper label="Debug Run ID">
    <IDElement>{trace.debugRunID}</IDElement>
  </ElementWrapper>
)}
```

### Import Changes

Add `IDElement` to the existing import from `../DetailsCard/Element`:

```tsx
// Current (lines 8-13):
import {
  CodeElement,
  ElementWrapper,
  LinkElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';

// Change to (add IDElement):
import {
  CodeElement,
  ElementWrapper,
  IDElement,
  LinkElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';
```

**Note**: If task 001 already added `PillElement` to this import, include it too.

### Type Reference

From `RunDetailsV4/types.ts`:
```typescript
export type Trace = {
  stepOp?: string | null;         // T1-11
  debugRunID?: string | null;     // T1-10
  // ... other fields
};
```

### Props Reference

`StepInfo` already has a `debug` prop (line 132-137):
```tsx
export const StepInfo = ({
  selectedStep,
  pollInterval: initialPollInterval,
  tracesPreviewEnabled,
  debug = false,   // <-- already exists, defaults to false
}: { ... }) => {
```

No prop changes needed.

## Embedded Spec Context

<details>
<summary>Relevant Requirements</summary>

**T1-11: Display Step Operation Type** (LOW priority)
- Data source: `trace.stepOp` — enum string, already fetched, used for bar styling but never rendered as text
- Map to human labels: RUN→"Step Run", INVOKE→"Invoke", SLEEP→"Sleep", WAIT_FOR_EVENT→"Wait for Event", AI_GATEWAY→"AI Gateway", WAIT_FOR_SIGNAL→"Wait for Signal"
- Fallback to raw value for unknown enums

**T1-10: Display Debug Run ID** (LOW priority)
- Data source: `trace.debugRunID` — ULID string, dev server only
- Render as monospace ID element when `debug === true` and non-null
- The `debug` prop already exists on `StepInfo` (defaults to `false`)
- No dev-server-ui file changes needed — shared `StepInfo` component handles both contexts

</details>

<details>
<summary>Architecture Context</summary>

- Both fields go in the expanded details section of `StepInfo`
- `stepOp` between Duration and `{stepKindInfo}`
- `debugRunID` after `{stepKindInfo}`, gated by `debug` prop
- Use `IDElement` for monospace debug run ID
- Use `TextElement` for step op label

</details>

<details>
<summary>Code Patterns</summary>

- Test framework: Vitest + @testing-library/react
- Use `data-testid` for DOM assertions (retro lesson)
- TypeScript strict mode
- Record<string, string> for enum label mapping with `??` fallback

</details>

## Retro Requirements

**Feature Retro File**: `specs/exe-1233/retro.md`
**Project Retro File**: `specs/retros.md`

### Before Starting (CRITICAL)

1. Read BOTH retro files for lessons from previous work
2. Apply **Stop** items: Don't couple tests to styling classes
3. Continue **Continue** practices: Use `data-testid` for DOM assertions; use existing helpers
4. Try **Start** approaches: Mock child components to test parent state

### After Implementation

If there are genuine takeaways, add them to `specs/exe-1233/retro.md`.
