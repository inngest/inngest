# Task exe-1233-001: Display Retry Attempt Badge

**Feature**: exe-1233
**Status**: pending
**Priority**: P1
**Depends on**: none

## Objective

Add a retry attempt badge to the V4 `StepInfo` header that shows the current attempt number when a step has been retried. Only display when `attempts > 0` (convert from 0-based to 1-based for human readability).

## Acceptance Criteria

- [ ] Tests written first (RED phase)
- [ ] Tests verified to fail before implementation
- [ ] Implementation passes all tests (GREEN phase)
- [ ] Badge renders "Attempt N" (1-based) when `trace.attempts > 0`
- [ ] Badge does NOT render when `trace.attempts === 0` or `trace.attempts === null`
- [ ] Badge uses `PillElement` for consistent styling
- [ ] Badge has `data-testid="retry-attempt-badge"` for test stability
- [ ] V3 components are NOT modified
- [ ] Code follows project conventions

## TDD Requirements

**Test File**: `ui/packages/components/src/RunDetailsV4/StepInfo.test.tsx` (create new)
**Test Command**: `cd ui/packages/components && pnpm test -- --run StepInfo`

## Project Tooling

**Lint**: `cd ui/packages/components && pnpm lint`
**Format**: `cd ui && pnpm run format`
**Build**: `cd ui/apps/dashboard && pnpm build`

### Tests to Write

1. **Badge renders when attempts > 0**: Render `StepInfo` with `trace.attempts = 2`. Assert element with `data-testid="retry-attempt-badge"` exists and contains text "Attempt 3".
2. **Badge hidden when attempts === 0**: Render `StepInfo` with `trace.attempts = 0`. Assert `data-testid="retry-attempt-badge"` is NOT in the document.
3. **Badge hidden when attempts === null**: Render `StepInfo` with `trace.attempts = null`. Assert `data-testid="retry-attempt-badge"` is NOT in the document.
4. **Badge shows correct 1-based number**: Render with `trace.attempts = 0, 1, 5`. Assert text shows "Attempt 1" (hidden), "Attempt 2", "Attempt 6" respectively.

**Testing notes**:
- Use `data-testid` attributes for DOM assertions (from retros: stable across styling changes)
- Mock child components (like `useGetTraceResult`, `useBooleanFlag`) to isolate `StepInfo`
- The `StepInfo` component requires these props:
  ```typescript
  {
    selectedStep: { trace: Trace; runID: string };
    pollInterval?: number;
    tracesPreviewEnabled?: boolean;
    debug?: boolean;
  }
  ```
- Mock `useGetTraceResult` to return `{ loading: false, data: null }`
- Mock `useBooleanFlag` to return `{ booleanFlag: () => ({ value: false, isReady: true }) }`
- Mock `usePathCreator` to return `{ pathCreator: { runPopout: () => '/run/123', function: () => '/fn/test', eventPopout: () => '/event/123' } }`
- Mock `useShared` to return `{ cloud: false }`

## Implementation Notes

**File to modify**: `ui/packages/components/src/RunDetailsV4/StepInfo.tsx`

### Current Code (lines 186-219)

The badge goes in the step header area, after the step name span and before the Rerun button:

```tsx
// Current code at lines 188-219:
<div className="flex min-h-11 w-full flex-row items-center justify-between border-none px-4">
  <div
    className="text-basis flex cursor-pointer items-center justify-start gap-2"
    onClick={() => setExpanded(!expanded)}
  >
    <RiArrowRightSLine
      className={`shrink-0 transition-transform duration-[250ms] ${
        expanded ? 'rotate-90' : ''
      }`}
    />

    <span className="text-basis text-sm font-normal">{trace.name}</span>
    {/* ADD BADGE HERE */}
  </div>
  {!debug && runID && trace.stepID && (!cloud || prettyInput) && (
    // ... Rerun button
  )}
</div>
```

### What to add

After line 199 (`<span className="text-basis text-sm font-normal">{trace.name}</span>`), add:

```tsx
{trace.attempts !== null && trace.attempts > 0 && (
  <PillElement data-testid="retry-attempt-badge" type="default">
    Attempt {trace.attempts + 1}
  </PillElement>
)}
```

**Note on PillElement**: Check if `PillElement` supports `data-testid` pass-through. It's defined in `ui/packages/components/src/DetailsCard/Element.tsx`:

```tsx
export function PillElement({
  children,
  type,
  appearance,
}: PillContentProps & { appearance?: PillAppearance }) {
  return (
    <Pill appearance={appearance}>
      <PillContent type={type}>{children}</PillContent>
    </Pill>
  );
}
```

If `PillElement` doesn't forward `data-testid`, wrap in a `<span data-testid="retry-attempt-badge">` instead.

### Import changes

Add `PillElement` to the existing import from `../DetailsCard/Element`:

```tsx
// Current (line 8-13):
import {
  CodeElement,
  ElementWrapper,
  LinkElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';

// Change to:
import {
  CodeElement,
  ElementWrapper,
  LinkElement,
  PillElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';
```

### Type reference

The `Trace` type (from `RunDetailsV4/types.ts`) already has `attempts`:
```typescript
export type Trace = {
  attempts: number | null;
  // ... other fields
};
```

No type changes needed.

## Embedded Spec Context

<details>
<summary>Relevant Requirements</summary>

**T1-1: Display Retry Attempt Count**

- Priority: HIGH
- Data source: `trace.attempts` — 0-based integer. Already in TypeScript type and GraphQL fragment.
- Where to render: V4 `StepInfo` right-panel, badge next to step name. Only when `attempts !== null && attempts > 0`.
- Display: "Attempt {attempts + 1}" (convert 0-based to 1-based)
- When `attempts === 0` or `null`, show nothing.

</details>

<details>
<summary>Architecture Context</summary>

- All changes target V4 only (`ui/packages/components/src/RunDetailsV4/`)
- Do NOT modify V3 (`ui/packages/components/src/RunDetailsV3/`)
- Use existing `PillElement` from `DetailsCard/Element.tsx` for badge styling
- Use `data-testid` attributes for test assertions (retro lesson)

</details>

<details>
<summary>Code Patterns</summary>

- Test framework: Vitest + @testing-library/react
- Test command: `cd ui/packages/components && pnpm test`
- TypeScript strict mode
- Use Tailwind CSS for styling
- Use `data-testid` for DOM assertions, not styling classes (retro lesson)

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
