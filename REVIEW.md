# Comprehensive Code Review: andy/EXE-1217-compound-bar (10 commits)

## Commit Sequence Confirmed

All 10 commits are present in the expected chronological order from `5c541d93a` through `d765764b3`.

---

## 1. CORRECTNESS

### 1a. Duplicated selection state between TimelineHeader and TimeBrush (High Severity)

**Files:** `ui/packages/components/src/RunDetailsV4/TimelineHeader.tsx` (lines 55-57) and `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx` (lines 58-59)

`TimelineHeader` maintains its own `selStart`/`selEnd` state (lines 55-56) and synchronizes it via the `onSelectionChange` callback from `TimeBrush`, which _also_ has its own `selectionStart`/`selectionEnd` state (lines 58-59). This creates two sources of truth for the same data. While it works because of the unidirectional callback flow, if `TimeBrush` ever batches or debounces updates, or if React batching defers a re-render, the two states could briefly diverge. The timestamp labels and split-color bar in `TimelineHeader` would then show stale positions relative to the actual handle positions in `TimeBrush`.

### 1b. onSelectionChange fires on mount via useEffect (Medium Severity)

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, lines 247-249

```typescript
useEffect(() => {
  onSelectionChange?.(selectionStart, selectionEnd);
}, [selectionStart, selectionEnd, onSelectionChange]);
```

This effect fires on _every_ render where `selectionStart` or `selectionEnd` changes, including the initial mount. It also fires if the parent re-renders and passes a new `onSelectionChange` reference (unless the parent memoizes it). In `TimelineHeader`, the parent does memoize `handleSelectionChange` with `useCallback` (line 59), but any consumer of `TimeBrush` that passes an inline arrow function will get an infinite loop: `onSelectionChange` changes -> effect fires -> parent sets state -> parent re-renders with new callback ref -> effect fires again.

### 1c. create-selection produces zero-width selection on click without drag (Low-Medium Severity)

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, lines 122-133

When the user clicks on the track (in default state or outside current selection) without dragging, `handleTrackMouseDown` sets both `selectionStart` and `selectionEnd` to the same `clickPercent`. Then `mouseup` fires immediately and no `mousemove` occurs. The result is a zero-width selection (e.g., `selectionStart === selectionEnd === 35`). Since `isDefaultSelection` checks for `start === 0 && end === 100`, this zero-width state is treated as "non-default," making the reset button appear and the split-bar render. But a zero-width selection is semantically meaningless and could produce visual artifacts (the left and right segments meet at the same point, middle segment has 0% width, two timestamp labels overlap at the same position).

### 1d. Hover line boundary condition uses strict comparison (Low Severity)

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, lines 151-152

```typescript
} else if (clampedPercent < selectionStart || clampedPercent > selectionEnd) {
```

The hover line is shown when the cursor is strictly outside the selection, but hidden when exactly at the boundary. The drag handles also sit at `selectionStart` and `selectionEnd`. Since floating-point percentages are computed from pixel positions, the exact boundary is unlikely to be hit, but this creates a theoretical dead zone at the exact selection edge where neither the cursor line nor the handle visual appears.

### 1e. Name span onClick competes with row onClick (Medium Severity)

**File:** `ui/packages/components/src/RunDetailsV4/TimelineBar.tsx`, lines 462-468

```typescript
onClick={
  expandable && !expanded
    ? () => {
        onToggle?.();
      }
    : undefined
}
```

The `<span>` for the name has an `onClick` handler that calls `onToggle()` when the row is expandable but collapsed. However, the parent `<div>` at line 428 also has `onClick={onClick}`. Since the click event bubbles, clicking the name text on a collapsed expandable row fires _both_ `onToggle()` (from the span) and `onClick()` (from the row). The span does not call `e.stopPropagation()`. This means a click on the name simultaneously expands the row AND triggers whatever the parent's `onClick` handler does (e.g., selection). This may or may not be intentional, but it is asymmetric with `ExpandToggle` (line 224) which _does_ call `e.stopPropagation()`.

### 1f. Selection highlight width uses leftWidth but highlight does not cover the right panel (Low Severity)

**File:** `ui/packages/components/src/RunDetailsV4/TimelineBar.tsx`, lines 434-438

```typescript
style={{
  left: `${indentPx - 4}px`,
  width: `calc(${leftWidth}% - ${indentPx - 4}px)`,
}}
```

The selection highlight only covers the left panel area (up to `leftWidth%`), not the bar visualization on the right side. This is a design choice, but if the intent is to highlight the entire row (as the old `bg-canvasSubtle` on the full row div did), it falls short. The highlight stops abruptly at the left panel boundary.

---

## 2. ARCHITECTURE

### 2a. TimelineHeader duplicates selection state that already exists in TimeBrush

As noted in 1a, `TimelineHeader` holds its own `selStart`/`selEnd` mirror of `TimeBrush`'s internal state. A cleaner pattern would be to either lift state out of `TimeBrush` entirely (making it controlled), or have `TimeBrush` accept a render prop / child component that receives the current selection values.

### 2b. TimeBrush uses refs for drag state but useState for selection

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`

`dragModeRef`, `dragStartXRef`, and `dragStartSelectionRef` are all `useRef`, while `selectionStart`/`selectionEnd` are `useState`. This mix is intentional (refs don't trigger re-renders during drag), but the `handleMouseMove` in the `useEffect` (line 169) reads `minSelectionWidth` from the closure. If `minSelectionWidth` ever changed during a drag, the effect would have a stale closure. Currently `minSelectionWidth` is the only dependency of the effect (line 244), so changing it mid-drag would create a new listener, causing the old drag to be lost. This is an unlikely edge case but represents a fragile coupling.

### 2c. Tight coupling between bar segment rendering in TimelineHeader and TimeBrush selection model

The split-color bar logic (default vs. 3-segment) in `TimelineHeader` (lines 143-172) is tightly embedded in the same component that does layout, markers, and timestamp labels. If other consumers need a split-color bar (e.g., a minimap), they would have to reimplement this logic.

---

## 3. TESTING

### 3a. Test selectors rely heavily on CSS class names (High Fragility)

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.test.tsx`

Throughout the tests, elements are located by complex CSS selectors like:
- Line 113: `.absolute.top-0.h-full`
- Line 156: `.absolute.top-0.h-full:not(.cursor-ew-resize)`
- Line 130: `.bg-surfaceMuted`
- Line 371: `.pointer-events-none.w-px`

These are extremely fragile. Any Tailwind class rename, ordering change, or refactoring of the markup would silently break these tests. The components use `data-testid` in some places (e.g., `time-brush-track`) but not consistently. The selection overlay, cursor line, and handle inner elements all lack test IDs.

### 3b. TimeBrush drag tests depend on mock getBoundingClientRect being set AFTER render

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.test.tsx`, lines 308-313

```typescript
function renderWithMock(props) {
  const result = render(<TimeBrush ... />);
  const outerContainer = result.container.firstChild as HTMLElement;
  outerContainer.getBoundingClientRect = vi.fn(() => ({ ...mockRect }));
  return { ...result, onSelectionChange, outerContainer };
}
```

The mock is applied _after_ render. If `TimeBrush` ever calls `getBoundingClientRect` during mount or during the first synchronous render cycle, the mock won't be in place. The current code only calls it in event handlers, so this works, but it is order-dependent.

### 3c. No test for minSelectionWidth enforcement during create-selection drag

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.test.tsx`, line 288-292

The `minSelectionWidth` describe block only verifies "it renders without error." There is no test asserting that dragging a handle to create a selection smaller than `minSelectionWidth` is actually prevented. The `create-selection` drag mode in the source code (lines 213-229) does _not_ enforce `minSelectionWidth` at all -- it allows any width including zero. Only the `left-handle` and `right-handle` modes enforce `minSelectionWidth`.

### 3d. Missing test: TimelineBar onClick does not fire when ExpandToggle is clicked

There is no test verifying that clicking the expand toggle button does NOT propagate to the row's `onClick`. The `ExpandToggle` calls `e.stopPropagation()`, but this behavior is not verified in tests.

### 3e. Missing test: expanded bar opacity for compound bars with segments

The opacity test (lines 195-206 of `TimelineBar.test.tsx`) only tests the simple bar path. When `segments` are provided, the compound bar container also receives `opacityStyle`, but there is no test for that path.

### 3f. TimelineHeader tests use a module-level mutable variable

**File:** `ui/packages/components/src/RunDetailsV4/TimelineHeader.test.tsx`, line 13

```typescript
let capturedOnSelectionChange: ((start: number, end: number) => void) | undefined;
```

This module-level mutable variable is never reset between tests. If test execution order changes (e.g., `.only`, parallel test runners), this captured callback could reference a stale component instance. The mock also does not reset `capturedOnSelectionChange` to `undefined` in `afterEach`.

### 3g. No test for the vertical guide line in TimelineBar

The commit `0910f38` added a vertical guide line for expanded areas (lines 508-517 of `TimelineBar.tsx`), but neither the existing tests nor the new tests verify that this element renders when `expandable && expanded` is true, or verify its positioning.

---

## 4. PERFORMANCE

### 4a. Cursor line causes re-render on every mouse move

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, lines 139-160

`handleTrackMouseMove` calls `setHoverPosition(clampedPercent)` on every `onMouseMove` event. Since `hoverPosition` is a `number` and floating-point values from pixel positions will almost always differ between events, this triggers a React re-render on every mouse move. Given that this also re-renders the `children` passed into `TimeBrush` (including the 3-segment bar), this could be expensive on low-end devices. No throttling, `requestAnimationFrame`, or debouncing is applied.

### 4b. Two separate state updates on each drag move in create-selection mode

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, lines 221-229

During the `create-selection` drag, both `setSelectionStart` and `setSelectionEnd` are called separately. While React 18's automatic batching within event handlers and effects will batch these together, the `handleMouseMove` is registered as a raw DOM event listener via `document.addEventListener` (not a React synthetic event), and in React 17 or certain edge cases, these could trigger two separate re-renders per mouse move.

### 4c. onSelectionChange effect fires on every selection state change

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, lines 247-249

During a drag, `selectionStart` and `selectionEnd` change on every mouse move, causing this effect to fire on every frame. The effect notifies the parent (`TimelineHeader`), which then sets its own state (`setSelStart`, `setSelEnd`), causing _another_ re-render cascade. This means each mouse move during a drag causes: TimeBrush re-render -> effect fires -> TimelineHeader re-render (timestamps + split bar).

### 4d. useMemo dependency array in VisualBar includes all view offset props

**File:** `ui/packages/components/src/RunDetailsV4/TimelineBar.tsx`, line 306

The `useMemo` for segment transformation depends on `[segments, originalBarStart, originalBarWidth, viewStartOffset, viewEndOffset]`. If any parent re-renders and creates a new `segments` array reference (even if contents are identical), the memo will recalculate. Since `segments` is an array prop, this is likely to happen on every parent re-render unless the parent memoizes the array.

---

## 5. ACCESSIBILITY

### 5a. TimeBrush drag handles have no ARIA attributes or keyboard support (High Severity)

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, lines 312-328 and 330-346

The left and right drag handles are plain `<div>` elements. They have no `role`, no `aria-label`, no `aria-valuemin`/`aria-valuemax`/`aria-valuenow`, no `tabIndex`, and no keyboard event handlers. Screen readers will not announce them. Keyboard users cannot tab to them or use arrow keys to adjust the selection. These should be `role="slider"` with proper ARIA attributes, or alternatively use the native HTML `<input type="range">` for each handle.

### 5b. TimeBrush selection area is not keyboard accessible

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, lines 282-295

The selection area is a `<div>` with `onMouseDown` handlers. There is no way for keyboard users to move the selection window.

### 5c. TimelineBar row click handler is on a div without role or keyboard interaction

**File:** `ui/packages/components/src/RunDetailsV4/TimelineBar.tsx`, line 428

```typescript
<div ... className="relative isolate flex h-7 cursor-pointer items-center" onClick={onClick} ...>
```

This clickable `<div>` has no `role="button"`, no `tabIndex`, and no `onKeyDown` handler. Keyboard users cannot select a row.

### 5d. TimelineBar name span click handler (line 462) has no keyboard equivalent

The `<span>` with the `onClick` for expanding collapsed rows cannot be activated via keyboard (no `tabIndex`, no `role`, no `onKeyDown`).

### 5e. TimeBrush Reset button uses `title` instead of `aria-label`

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, line 264

The reset button uses `title="Reset selection"` but does not have an `aria-label`. The visible text is just "Reset" which is ambiguous. `title` is not reliably announced by all screen readers.

### 5f. Timestamp labels are `pointer-events-none` divs with no semantic role

**File:** `ui/packages/components/src/RunDetailsV4/TimelineHeader.tsx`, lines 107-138

The timestamp labels above drag handles are plain divs with `pointer-events-none`. They have no `aria-live` or `role` attributes, so screen reader users will not be notified when the selection changes and new timestamps appear.

---

## 6. CSS / STYLING

### 6a. z-index values are spread across components without a scale

The timestamp labels use `z-20` (TimelineHeader.tsx line 108), the cursor line uses `zIndex: 10` as inline style (TimeBrush.tsx line 307), and the selection highlight uses `-z-10` with `isolate` (TimelineBar.tsx lines 427, 434). There is no centralized z-index scale or documentation. This creates risk of future stacking order conflicts.

### 6b. Track extends outside its container with negative positioning

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, line 275

```typescript
className="absolute inset-0 -bottom-2 -top-6"
```

The track area extends 24px above (`-top-6` = `-1.5rem`) and 8px below the brush. This overflow may conflict with other elements in the parent layout, especially the timestamp labels which are positioned at `top: -18px`. The track and the timestamp labels could overlap, creating unexpected click targets (the track's `onMouseDown` might intercept clicks intended for other nearby UI).

### 6c. Cursor line extends beyond the brush container

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, line 303

```typescript
className={cn('pointer-events-none absolute -top-6 bottom-0 w-px', cursorLineClassName)}
```

The cursor line also extends `-top-6` above the brush. If the parent has `overflow: hidden`, this line would be clipped. If not, it visually extends into the timestamp/marker area, which may be intentional but creates visual coupling between components that should be independent.

### 6d. Selection highlight uses calc() mixing percentage and px units

**File:** `ui/packages/components/src/RunDetailsV4/TimelineBar.tsx`, line 437

```typescript
width: `calc(${leftWidth}% - ${indentPx - 4}px)`,
```

This works but means the highlight width depends on both the percentage-based `leftWidth` and pixel-based indentation. If `indentPx - 4` is negative (when depth is 0, `indentPx` is 4, so `indentPx - 4 = 0`), the calc reduces to just `leftWidth%`. But at depth 0, `left` is `0px`, and the highlight spans the full left panel width. The `-4` offset is a magic number not tied to any constant.

### 6e. `left: 0` used as number instead of string in bar-segment-left

**File:** `ui/packages/components/src/RunDetailsV4/TimelineHeader.tsx`, line 156

```typescript
style={{ left: 0, width: `${selStart}%` }}
```

Using `left: 0` as a bare number works in React (it is treated as `0px`), but it is inconsistent with the other segments which use string values like `` left: `${selStart}%` ``.

---

## 7. CODE QUALITY

### 7a. Magic number 8 in vertical guide line positioning

**File:** `ui/packages/components/src/RunDetailsV4/TimelineBar.tsx`, line 512

```typescript
left: `${indentPx + 8}px`,
```

The `8` is not tied to any named constant. It appears to be half the width of the expand toggle icon (16px arrow), but this relationship is not documented and would break if the icon size changed.

### 7b. Magic number -4 in selection highlight

**File:** `ui/packages/components/src/RunDetailsV4/TimelineBar.tsx`, line 436

```typescript
left: `${indentPx - 4}px`,
```

The `4` appears to match `TIMELINE_CONSTANTS.BASE_LEFT_PADDING_PX`, but instead of referencing the constant, a raw `4` is used. This creates a maintenance risk if the constant changes.

### 7c. Inconsistent comment style and feature references

Some files reference "Feature: 001-composable-timeline-bar" in their headers, some reference "EXE-1217, Task 001" (TimelineBar.test.tsx line 193), some reference "Task 005" (TimeBrush.test.tsx line 295), and some reference "FR-002, FR-003, FR-005" (TimeBrush.test.tsx line 125). These appear to be from different tracking systems and feature specs. A consistent referencing convention would improve traceability.

### 7d. Duplicate test for reset button not showing in default state

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.test.tsx`

Lines 32-36 ("does not show reset button when at default selection") and lines 62-67 (same description) are effectively identical tests in different `describe` blocks.

### 7e. The `VisualBar` function component uses `useMemo` but is not memoized itself

**File:** `ui/packages/components/src/RunDetailsV4/TimelineBar.tsx`, line 243

`VisualBar` is a nested function component defined outside of `TimelineBar` (which is good), and it uses `useMemo` internally for segment transformation. However, `VisualBar` itself is not wrapped in `React.memo`, so it re-renders whenever `TimelineBar` re-renders, even if all its props are identical. The internal `useMemo` only helps with the segment calculation, not the overall render.

### 7f. `expanded` prop is passed to `VisualBar` as `expandable && expanded`

**File:** `ui/packages/components/src/RunDetailsV4/TimelineBar.tsx`, line 495

```typescript
expanded={expandable && expanded}
```

This means `expanded` on `VisualBar` can be `boolean | undefined` (since `expandable` can be `undefined`). TypeScript treats `undefined && false` as `undefined`, which is falsy, so it works, but the typing is imprecise. The `VisualBar` prop type is `expanded?: boolean`.

### 7g. `handleTrackMouseLeave` is a separate callback that only sets null

**File:** `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx`, lines 163-165

```typescript
const handleTrackMouseLeave = useCallback(() => {
  setHoverPosition(null);
}, []);
```

This is wrapped in `useCallback` with an empty dependency array. Since `setHoverPosition` is a state setter (stable reference), this is correct, but `useCallback` with an empty deps array and no captures is effectively the same as defining the function at module scope. The overhead of `useCallback` here is negligible but also unnecessary.

---

## Summary of Key Findings by Priority

**Should address before merge:**
- (1e) Name span `onClick` does not stop propagation, causing dual fire with row `onClick`
- (1c) Zero-width selection on click-without-drag -- no minimum enforcement in `create-selection` mode
- (3c) `minSelectionWidth` is not enforced for `create-selection` drag, contradicting the prop's implied contract

**Should address soon after merge:**
- (5a) Drag handles are completely inaccessible to keyboard/screen reader users
- (5c) Clickable row divs need keyboard support
- (4a/4c) Re-render cascade on every mouse move during drag (cursor line + parent notification)
- (3a) Tests rely on CSS class selectors instead of `data-testid`

**Worth noting / lower priority:**
- (1a) Duplicated selection state between TimelineHeader and TimeBrush
- (6a) No centralized z-index scale
- (7a/7b) Magic numbers in positioning calculations
- (3f) Module-level mutable variable in test file not reset between tests
- (3g) No test for vertical guide line
