# Task exe-1233-003: Add Matched Event ID to WaitInfo

**Feature**: exe-1233
**Status**: pending
**Priority**: P1
**Depends on**: none

## Objective

Add the matched event ID field (T1-4) to the `WaitInfo` component in V4 `StepInfo.tsx`. When a `step.waitForEvent()` matches an event, show the `foundEventID` as a clickable link. Hide when null (still waiting or timed out).

## Acceptance Criteria

- [ ] Tests written first (RED phase)
- [ ] Tests verified to fail before implementation
- [ ] Implementation passes all tests (GREEN phase)
- [ ] `foundEventID` renders as a link when non-null using `pathCreator.eventPopout()`
- [ ] Field is hidden when `foundEventID === null`
- [ ] Uses `ElementWrapper` + `LinkElement` pattern
- [ ] `usePathCreator` hook added to `WaitInfo` component
- [ ] V3 components are NOT modified
- [ ] Code follows project conventions

## TDD Requirements

**Test File**: `ui/packages/components/src/RunDetailsV4/StepInfo.test.tsx` (add to existing from tasks 001/002)
**Test Command**: `cd ui/packages/components && pnpm test -- --run StepInfo`

## Project Tooling

**Lint**: `cd ui/packages/components && pnpm lint`
**Format**: `cd ui && pnpm run format`
**Build**: `cd ui/apps/dashboard && pnpm build`

### Tests to Write

1. **Matched Event renders when foundEventID is non-null**: Render `StepInfo` with a wait-type `stepInfo` where `foundEventID = "01MATCHED..."`. Assert "Matched Event" label and link are present, link href matches `pathCreator.eventPopout({ eventID: "01MATCHED..." })`.
2. **Matched Event hidden when foundEventID is null**: Render with `foundEventID = null`. Assert no "Matched Event" label in document.
3. **Existing WaitInfo fields still render**: Assert "Event name", "Timeout", "Timed out", "Match expression" fields still appear.

**Testing notes**:
- `WaitInfo` is an internal component. Test via `StepInfo` with a wait-type `stepInfo`.
- To trigger `WaitInfo`, set `trace.stepInfo` to match `StepInfoWait` type (must have `foundEventID` field).
- Mock `usePathCreator` same as task 002.
- Test data for a wait-type stepInfo:
  ```typescript
  const waitStepInfo: StepInfoWait = {
    eventName: 'test/event',
    expression: 'event.data.id == async.data.id',
    timeout: '2026-01-01T00:00:00Z',
    foundEventID: '01MATCHED123',  // or null for hidden test
    timedOut: false,
  };
  ```

## Implementation Notes

**File to modify**: `ui/packages/components/src/RunDetailsV4/StepInfo.tsx`

### Current WaitInfo Component (lines 76-98)

```tsx
const WaitInfo = ({ stepInfo }: { stepInfo: StepInfoWait }) => {
  const timeout = toMaybeDate(stepInfo.timeout);
  return (
    <>
      <ElementWrapper label="Event name">
        <TextElement>{stepInfo.eventName}</TextElement>
      </ElementWrapper>
      <ElementWrapper label="Timeout">
        {timeout ? <TimeElement date={timeout} /> : <TextElement>-</TextElement>}
      </ElementWrapper>
      <ElementWrapper label="Timed out">
        <TextElement>{maybeBooleanToString(stepInfo.timedOut)}</TextElement>
      </ElementWrapper>
      <ElementWrapper className="w-full" label="Match expression">
        {stepInfo.expression ? (
          <CodeElement value={stepInfo.expression} />
        ) : (
          <TextElement>-</TextElement>
        )}
      </ElementWrapper>
    </>
  );
};
```

### Updated WaitInfo Component

```tsx
const WaitInfo = ({ stepInfo }: { stepInfo: StepInfoWait }) => {
  const { pathCreator } = usePathCreator();  // NEW: add hook
  const timeout = toMaybeDate(stepInfo.timeout);
  return (
    <>
      <ElementWrapper label="Event name">
        <TextElement>{stepInfo.eventName}</TextElement>
      </ElementWrapper>
      <ElementWrapper label="Timeout">
        {timeout ? <TimeElement date={timeout} /> : <TextElement>-</TextElement>}
      </ElementWrapper>
      <ElementWrapper label="Timed out">
        <TextElement>{maybeBooleanToString(stepInfo.timedOut)}</TextElement>
      </ElementWrapper>
      <ElementWrapper className="w-full" label="Match expression">
        {stepInfo.expression ? (
          <CodeElement value={stepInfo.expression} />
        ) : (
          <TextElement>-</TextElement>
        )}
      </ElementWrapper>
      {/* T1-4: Matched Event ID */}
      {stepInfo.foundEventID && (
        <ElementWrapper label="Matched Event">
          <LinkElement href={pathCreator.eventPopout({ eventID: stepInfo.foundEventID })}>
            {stepInfo.foundEventID}
          </LinkElement>
        </ElementWrapper>
      )}
    </>
  );
};
```

### Key Change: Add usePathCreator to WaitInfo

`WaitInfo` currently does NOT use `usePathCreator`. You must add the hook call:
```tsx
const { pathCreator } = usePathCreator();
```

`usePathCreator` is already imported at the top of `StepInfo.tsx` (line 18):
```tsx
import { usePathCreator } from '../SharedContext/usePathCreator';
```

No new imports needed — just add the hook call inside `WaitInfo`.

### Type Reference

`StepInfoWait` from `RunDetailsV4/types.ts`:
```typescript
export type StepInfoWait = {
  eventName: string;
  expression: string | null;
  timeout: string;
  foundEventID: string | null;   // T1-4: null when waiting or timed out
  timedOut: boolean | null;
};
```

### PathCreator Route

```typescript
eventPopout: (params: { eventID: string }) => Route;
```

## Embedded Spec Context

<details>
<summary>Relevant Requirements</summary>

**T1-4: Display Matched Event ID for WaitForEvent** (MEDIUM priority)
- Data source: `trace.stepInfo.foundEventID` on `WaitForEventStepInfo`
- Null when still waiting or timed out
- Render as link using `pathCreator.eventPopout({ eventID })` when non-null
- Hide completely when null

</details>

<details>
<summary>Architecture Context</summary>

- Modify only `WaitInfo` component in `RunDetailsV4/StepInfo.tsx`
- Add `usePathCreator` hook to `WaitInfo` (currently missing)
- Use `LinkElement` for the event ID link
- Conditional rendering: `{stepInfo.foundEventID && (...)}`

</details>

<details>
<summary>Code Patterns</summary>

- Test framework: Vitest + @testing-library/react
- Use `data-testid` for DOM assertions (retro lesson)
- TypeScript strict mode

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
