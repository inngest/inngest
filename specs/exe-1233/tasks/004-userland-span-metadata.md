# Task exe-1233-004: Display Userland Span Metadata in UserlandAttrs

**Feature**: exe-1233
**Status**: pending
**Priority**: P2
**Depends on**: none

## Objective

Extend the V4 `UserlandAttrs` component to display OTEL span metadata fields (T1-5 through T1-8): `spanName`, `spanKind`, `serviceName`, `scopeName`, `scopeVersion`, and `resourceAttrs`. Currently only `spanAttrs` is rendered. Add a metadata header section above the existing attributes table and a resource attributes table below.

## Acceptance Criteria

- [ ] Tests written first (RED phase)
- [ ] Tests verified to fail before implementation
- [ ] Implementation passes all tests (GREEN phase)
- [ ] `spanName` renders when non-null with label "Span"
- [ ] `spanKind` renders as a badge when non-null with label "Kind"
- [ ] `serviceName` renders when non-null with label "Service"
- [ ] `scopeName` + `scopeVersion` render together when either is non-null with label "Scope"
- [ ] `resourceAttrs` JSON is parsed and rendered as key-value table when non-null
- [ ] `resourceAttrs` internal prefixes are filtered (same as `spanAttrs`)
- [ ] Malformed `resourceAttrs` JSON: catch error, `console.info`, don't render section
- [ ] All null fields are completely hidden (no empty labels)
- [ ] Component still renders `null` when all data is absent
- [ ] V3 components are NOT modified
- [ ] Code follows project conventions

## TDD Requirements

**Test File**: `ui/packages/components/src/RunDetailsV4/UserlandAttrs.test.tsx` (create new)
**Test Command**: `cd ui/packages/components && pnpm test -- --run UserlandAttrs`

## Project Tooling

**Lint**: `cd ui/packages/components && pnpm lint`
**Format**: `cd ui && pnpm run format`
**Build**: `cd ui/apps/dashboard && pnpm build`

### Tests to Write

1. **spanName renders when non-null**: Render with `userlandSpan.spanName = "HTTP GET /api/users"`. Assert "Span" label and text "HTTP GET /api/users" present.
2. **spanKind renders as badge when non-null**: Render with `userlandSpan.spanKind = "CLIENT"`. Assert "Kind" label and "CLIENT" text present.
3. **serviceName renders when non-null**: Render with `userlandSpan.serviceName = "user-service"`. Assert "Service" label and text present.
4. **scope renders with name and version**: Render with `scopeName = "my-scope"`, `scopeVersion = "1.0.0"`. Assert "Scope" label with text containing both.
5. **scope renders with only name**: Render with `scopeName = "my-scope"`, `scopeVersion = null`. Assert "Scope" label with text "my-scope".
6. **resourceAttrs renders as key-value table**: Render with `resourceAttrs = '{"host.name":"server-1","service.version":"2.0"}'`. Assert table rows with keys "host.name" and "service.version".
7. **resourceAttrs filters internal prefixes**: Render with `resourceAttrs = '{"sys.internal":"hidden","host.name":"visible"}'`. Assert "sys.internal" NOT in document, "host.name" present.
8. **resourceAttrs malformed JSON handled gracefully**: Render with `resourceAttrs = "not-valid-json"`. Assert no resource attrs section rendered, no error thrown.
9. **All null fields render nothing**: Render with all metadata fields null and `spanAttrs = '{"key":"value"}'`. Assert no "Span", "Kind", "Service", "Scope" labels, but spanAttrs table still renders.
10. **Component returns null when all data is absent**: Render with all fields null. Assert component renders nothing.

**Testing notes**:
- `UserlandAttrs` is a simple exported component. It can be tested directly.
- Props: `{ userlandSpan: UserlandSpanType }`
- Test data template:
  ```typescript
  const baseUserlandSpan: UserlandSpanType = {
    spanName: null,
    spanKind: null,
    serviceName: null,
    scopeName: null,
    scopeVersion: null,
    spanAttrs: null,
    resourceAttrs: null,
  };
  ```

## Implementation Notes

**File to modify**: `ui/packages/components/src/RunDetailsV4/UserlandAttrs.tsx`

### Current Code (full file, 39 lines)

```tsx
import type { UserlandSpanType } from './types';

const internalPrevixes = ['sys', 'inngest', 'userland', 'sdk'];

export const UserlandAttrs = ({ userlandSpan }: { userlandSpan: UserlandSpanType }) => {
  let attrs = null;

  try {
    attrs = userlandSpan.spanAttrs && JSON.parse(userlandSpan.spanAttrs);
  } catch (error) {
    console.info('Error parsing userland span attributes', error);
  }

  return attrs ? (
    <div className="h-full overflow-y-auto">
      <div className="mb-4 mt-2 flex max-h-full flex-col gap-2">
        <div className="text-muted bg-canvasSubtle sticky top-0 flex flex-row px-4 py-2 text-sm font-medium leading-tight">
          <div className="w-72">Key</div>
          <div className="">Value</div>
        </div>
        {Object.entries(attrs)
          .filter(([key]) => !internalPrevixes.some((prefix) => key.startsWith(prefix)))
          .map(([key, value]) => {
            return (
              <div
                key={`userland-span-attr-${key}`}
                className="border-canvasSubtle flex flex-row items-center border-b px-4 pb-2"
              >
                <div className="text-muted w-72 text-sm font-normal leading-tight">{key}</div>
                <div className="text-basis truncate text-sm font-normal leading-tight">
                  {String(value) || '--'}
                </div>
              </div>
            );
          })}
      </div>
    </div>
  ) : null;
};
```

### Implementation Approach

1. **Parse `resourceAttrs`** using the same try/catch pattern as `spanAttrs`:
   ```tsx
   let resourceAttrsObj = null;
   try {
     resourceAttrsObj = userlandSpan.resourceAttrs && JSON.parse(userlandSpan.resourceAttrs);
   } catch (error) {
     console.info('Error parsing userland resource attributes', error);
   }
   ```

2. **Check if there's any metadata to show**:
   ```tsx
   const hasMetadata = userlandSpan.spanName || userlandSpan.spanKind ||
     userlandSpan.serviceName || userlandSpan.scopeName || userlandSpan.scopeVersion;
   const hasContent = attrs || resourceAttrsObj || hasMetadata;
   if (!hasContent) return null;
   ```

3. **Add metadata header section** above the existing key-value table:
   ```tsx
   {hasMetadata && (
     <div className="flex flex-row flex-wrap gap-x-10 gap-y-2 px-4 py-2">
       {userlandSpan.spanName && (
         <div className="text-sm">
           <dt className="text-muted text-xs">Span</dt>
           <dd className="text-basis">{userlandSpan.spanName}</dd>
         </div>
       )}
       {userlandSpan.spanKind && (
         <div className="text-sm">
           <dt className="text-muted text-xs">Kind</dt>
           <dd><PillElement type="default">{userlandSpan.spanKind}</PillElement></dd>
         </div>
       )}
       {userlandSpan.serviceName && (
         <div className="text-sm">
           <dt className="text-muted text-xs">Service</dt>
           <dd className="text-basis">{userlandSpan.serviceName}</dd>
         </div>
       )}
       {(userlandSpan.scopeName || userlandSpan.scopeVersion) && (
         <div className="text-sm">
           <dt className="text-muted text-xs">Scope</dt>
           <dd className="text-basis">
             {[userlandSpan.scopeName, userlandSpan.scopeVersion].filter(Boolean).join(' ')}
           </dd>
         </div>
       )}
     </div>
   )}
   ```

4. **Add `resourceAttrs` table** below `spanAttrs` table using the same key-value format and same `internalPrevixes` filtering.

5. **Use `data-testid` attributes** on key sections for test stability:
   - `data-testid="span-metadata-header"` on the metadata header
   - `data-testid="resource-attrs-table"` on the resource attrs section

### Type Reference

```typescript
export type UserlandSpanType = {
  spanName: string | null;        // T1-5
  spanKind: string | null;        // T1-6
  serviceName: string | null;     // T1-7
  scopeName: string | null;       // T1-8
  scopeVersion: string | null;    // T1-8
  spanAttrs: string | null;       // already rendered
  resourceAttrs: string | null;   // T1-8
};
```

### Import Changes

If using `PillElement` for `spanKind` badge, add import:
```tsx
import { PillElement } from '../DetailsCard/Element';
```

Or use a simple inline badge with Tailwind classes to avoid adding a new import. Follow whichever approach feels cleaner — `PillElement` is more consistent with `StepInfo.tsx` patterns but adds a dependency; an inline `<span className="...">` is simpler.

## Embedded Spec Context

<details>
<summary>Relevant Requirements</summary>

**T1-5 through T1-8: Display Userland Span Fields** (LOW priority)
- `spanName`: Display when non-null
- `spanKind`: Display as badge (CLIENT/SERVER/INTERNAL/PRODUCER/CONSUMER)
- `serviceName`: Display when non-null
- `scopeName` + `scopeVersion`: Display together when either non-null
- `resourceAttrs`: JSON string → parse → key-value table with same prefix filtering as `spanAttrs`
- Error handling for `resourceAttrs`: follow `spanAttrs` pattern (catch → console.info → skip section)

</details>

<details>
<summary>Architecture Context</summary>

- Modify only `UserlandAttrs.tsx` in `RunDetailsV4/`
- JSON parse pattern: `try { JSON.parse() } catch { console.info(); }` → render nothing on failure
- Internal prefix filter: `['sys', 'inngest', 'userland', 'sdk']` (already defined as `internalPrevixes`)
- Metadata header goes above existing `spanAttrs` table
- `resourceAttrs` table goes below existing `spanAttrs` table

</details>

<details>
<summary>Code Patterns</summary>

- Test framework: Vitest + @testing-library/react
- Use `data-testid` for DOM assertions (retro lesson)
- TypeScript strict mode
- Tailwind CSS for styling

</details>

## Retro Requirements

**Feature Retro File**: `specs/exe-1233/retro.md`
**Project Retro File**: `specs/retros.md`

### Before Starting (CRITICAL)

1. Read BOTH retro files for lessons from previous work
2. Apply **Stop** items: Don't couple tests to styling classes
3. Continue **Continue** practices: Use `data-testid` for DOM assertions; use existing helpers
4. Try **Start** approaches: Extract duplicated computations during refactor phase

### After Implementation

If there are genuine takeaways, add them to `specs/exe-1233/retro.md`.
