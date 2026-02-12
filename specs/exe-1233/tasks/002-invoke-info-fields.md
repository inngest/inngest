# Task exe-1233-002: Add InvokeInfo Fields (Function ID, Triggering Event, Return Event)

**Feature**: exe-1233
**Status**: pending
**Priority**: P1
**Depends on**: none

## Objective

Add three new fields to the `InvokeInfo` component in V4 `StepInfo.tsx`: the invoked function ID (T1-2), triggering event ID (T1-3), and return event ID (T1-9). All three are already fetched via GraphQL but not rendered. Function ID and triggering event ID always display; return event ID only displays when non-null.

## Acceptance Criteria

- [ ] Tests written first (RED phase)
- [ ] Tests verified to fail before implementation
- [ ] Implementation passes all tests (GREEN phase)
- [ ] `functionID` renders as a link to the function page using `pathCreator.function()`
- [ ] `triggeringEventID` renders as a link to the event page using `pathCreator.eventPopout()`
- [ ] `returnEventID` renders as a link when non-null, hidden when null
- [ ] All three use `ElementWrapper` + `LinkElement` pattern
- [ ] Field order: Function, Triggering Event, Run, Timeout, Timed out, Return Event
- [ ] V3 components are NOT modified
- [ ] Code follows project conventions

## TDD Requirements

**Test File**: `ui/packages/components/src/RunDetailsV4/StepInfo.test.tsx` (create new or add to existing from task 001)
**Test Command**: `cd ui/packages/components && pnpm test -- --run StepInfo`

## Project Tooling

**Lint**: `cd ui/packages/components && pnpm lint`
**Format**: `cd ui && pnpm run format`
**Build**: `cd ui/apps/dashboard && pnpm build`

### Tests to Write

1. **Function ID renders as link**: Render `InvokeInfo` with `stepInfo.functionID = "my-app-send-email"`. Assert a link exists with text "my-app-send-email" and `href` matching `pathCreator.function({ functionSlug: "my-app-send-email" })`.
2. **Triggering Event ID renders as link**: Render with `stepInfo.triggeringEventID = "01ABCDEF..."`. Assert a link exists with text containing the event ID and `href` matching `pathCreator.eventPopout({ eventID: "01ABCDEF..." })`.
3. **Return Event ID renders when non-null**: Render with `stepInfo.returnEventID = "01RETURN..."`. Assert "Return Event" label and link are present.
4. **Return Event ID hidden when null**: Render with `stepInfo.returnEventID = null`. Assert no "Return Event" label in document.
5. **All fields have correct labels**: Assert label text "Function", "Triggering Event", "Return Event" via `data-testid` or label text queries.

**Testing notes**:
- `InvokeInfo` is an internal component of `StepInfo.tsx` (not exported). Test via `StepInfo` with an invoke-type `stepInfo`.
- To trigger `InvokeInfo`, set `trace.stepInfo` to match `StepInfoInvoke` type (must have `triggeringEventID` field).
- Mock `usePathCreator` to return predictable URLs:
  ```typescript
  vi.mock('../SharedContext/usePathCreator', () => ({
    usePathCreator: () => ({
      pathCreator: {
        function: ({ functionSlug }: { functionSlug: string }) => `/functions/${functionSlug}`,
        eventPopout: ({ eventID }: { eventID: string }) => `/events/${eventID}`,
        runPopout: ({ runID }: { runID: string }) => `/runs/${runID}`,
      },
    }),
  }));
  ```
- Mock `useGetTraceResult`, `useBooleanFlag`, `useShared` as in task 001.

## Implementation Notes

**File to modify**: `ui/packages/components/src/RunDetailsV4/StepInfo.tsx`

### Current InvokeInfo Component (lines 43-65)

```tsx
const InvokeInfo = ({ stepInfo }: { stepInfo: StepInfoInvoke }) => {
  const { pathCreator } = usePathCreator();
  const timeout = toMaybeDate(stepInfo.timeout);
  return (
    <>
      <ElementWrapper label="Run">
        {stepInfo.runID ? (
          <LinkElement href={pathCreator.runPopout({ runID: stepInfo.runID })}>
            {stepInfo.runID}
          </LinkElement>
        ) : (
          '-'
        )}
      </ElementWrapper>
      <ElementWrapper label="Timeout">
        {timeout ? <TimeElement date={timeout} /> : <TextElement>-</TextElement>}
      </ElementWrapper>
      <ElementWrapper label="Timed out">
        <TextElement>{maybeBooleanToString(stepInfo.timedOut)}</TextElement>
      </ElementWrapper>
    </>
  );
};
```

### Updated InvokeInfo Component

```tsx
const InvokeInfo = ({ stepInfo }: { stepInfo: StepInfoInvoke }) => {
  const { pathCreator } = usePathCreator();
  const timeout = toMaybeDate(stepInfo.timeout);
  return (
    <>
      {/* T1-2: Function ID */}
      <ElementWrapper label="Function">
        <LinkElement href={pathCreator.function({ functionSlug: stepInfo.functionID })}>
          {stepInfo.functionID}
        </LinkElement>
      </ElementWrapper>
      {/* T1-3: Triggering Event ID */}
      <ElementWrapper label="Triggering Event">
        <LinkElement href={pathCreator.eventPopout({ eventID: stepInfo.triggeringEventID })}>
          {stepInfo.triggeringEventID}
        </LinkElement>
      </ElementWrapper>
      {/* Existing: Run */}
      <ElementWrapper label="Run">
        {stepInfo.runID ? (
          <LinkElement href={pathCreator.runPopout({ runID: stepInfo.runID })}>
            {stepInfo.runID}
          </LinkElement>
        ) : (
          '-'
        )}
      </ElementWrapper>
      {/* Existing: Timeout */}
      <ElementWrapper label="Timeout">
        {timeout ? <TimeElement date={timeout} /> : <TextElement>-</TextElement>}
      </ElementWrapper>
      {/* Existing: Timed out */}
      <ElementWrapper label="Timed out">
        <TextElement>{maybeBooleanToString(stepInfo.timedOut)}</TextElement>
      </ElementWrapper>
      {/* T1-9: Return Event ID (conditional) */}
      {stepInfo.returnEventID && (
        <ElementWrapper label="Return Event">
          <LinkElement href={pathCreator.eventPopout({ eventID: stepInfo.returnEventID })}>
            {stepInfo.returnEventID}
          </LinkElement>
        </ElementWrapper>
      )}
    </>
  );
};
```

### Type Reference

`StepInfoInvoke` already has all fields (from `RunDetailsV4/types.ts`):
```typescript
export type StepInfoInvoke = {
  triggeringEventID: string;      // T1-3: always present
  functionID: string;             // T1-2: always present
  returnEventID: string | null;   // T1-9: null when still waiting
  runID: string | null;           // already rendered
  timeout: string;                // already rendered
  timedOut: boolean | null;       // already rendered
};
```

### PathCreator Routes Available

From `ui/packages/components/src/SharedContext/usePathCreator.ts`:
```typescript
export type PathCreator = {
  function: (params: { functionSlug: string }) => Route;      // T1-2
  eventPopout: (params: { eventID: string }) => Route;        // T1-3, T1-9
  runPopout: (params: { runID: string }) => Route;            // already used
  // ... other routes
};
```

`usePathCreator` is already imported and used in `InvokeInfo` — no new imports needed for pathCreator.

### No Import Changes Needed

All elements used (`ElementWrapper`, `LinkElement`, `TextElement`, `TimeElement`) are already imported in `StepInfo.tsx`.

## Embedded Spec Context

<details>
<summary>Relevant Requirements</summary>

**T1-2: Display Invoked Function ID** (HIGH priority)
- Data source: `trace.stepInfo.functionID` on `InvokeStepInfo`
- Render as link using `pathCreator.function({ functionSlug: stepInfo.functionID })`
- Always present (non-nullable `string`)

**T1-3: Display Triggering Event ID** (MEDIUM priority)
- Data source: `trace.stepInfo.triggeringEventID` on `InvokeStepInfo`
- Render as link using `pathCreator.eventPopout({ eventID: stepInfo.triggeringEventID })`
- Always present (non-nullable `string`)

**T1-9: Display Return Event ID** (LOW priority)
- Data source: `trace.stepInfo.returnEventID` on `InvokeStepInfo`
- Render as link using `pathCreator.eventPopout()` — only when non-null
- Null when invocation is still in progress

</details>

<details>
<summary>Architecture Context</summary>

- All changes target V4 only (`RunDetailsV4/StepInfo.tsx`)
- Use `LinkElement` + `pathCreator.*` for all ID links (routes confirmed to exist)
- `InvokeInfo` already uses `usePathCreator` — no new hook calls needed
- Follow existing field ordering pattern: metadata first, then existing fields

</details>

<details>
<summary>Code Patterns</summary>

- Test framework: Vitest + @testing-library/react
- Use `data-testid` for DOM assertions (retro lesson)
- Mock child components/hooks to isolate component (retro lesson)
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
