# RunDetailsV4 Timeline System

A composable, extensible timeline visualization system built with a "one-bar-to-rule-them-all" architecture.

## Overview

This timeline system renders hierarchical, time-based data with support for:

- **Infinite nesting depth** - Recursive rendering handles any level of hierarchy
- **Interactive zooming** - Time Brush allows selecting and zooming into timeline regions
- **Compound bars** - Single bars can contain multiple styled segments
- **Timing breakdowns** - Expandable bars reveal queue vs. execution time
- **Consistent styling** - Style keys ensure visual consistency across bar types

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ Timeline (container)                                            │
│  ├─ TimelineHeader (markers + TimeBrush)                        │
│  │    └─ TimeBrush (reusable range selection)                   │
│  └─ TimelineBarRenderer[] (recursive)                           │
│       └─ TimelineBar (single row)                               │
│            ├─ Left Panel: icon, name, duration, expand toggle   │
│            └─ Right Panel: VisualBar (positioned bar segments)  │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### TimelineBar

The core building block. Renders a single timeline row with:

- **Left panel**: Name, optional icon, duration, expand/collapse toggle
- **Right panel**: Visual bar positioned by percentage, with optional segments

```tsx
<TimelineBar
  name="Process Data"
  duration={1500}
  startPercent={10}
  widthPercent={30}
  depth={0}
  leftWidth={40}
  style="step.run"
  expandable
  expanded={isExpanded}
  onToggle={() => toggle()}
>
  {/* Nested children rendered when expanded */}
</TimelineBar>
```

#### Props

| Prop              | Type           | Description                             |
| ----------------- | -------------- | --------------------------------------- |
| `id`              | `string`       | Unique identifier                       |
| `name`            | `string`       | Display name in left panel              |
| `duration`        | `number`       | Duration in milliseconds                |
| `icon`            | `BarIcon`      | Optional icon (overrides style default) |
| `startPercent`    | `number`       | Start position (0-100)                  |
| `widthPercent`    | `number`       | Width (0-100)                           |
| `depth`           | `number`       | Nesting depth (affects indentation)     |
| `leftWidth`       | `number`       | Left panel width percentage             |
| `style`           | `BarStyleKey`  | Visual style key                        |
| `segments`        | `BarSegment[]` | Optional compound bar segments          |
| `expandable`      | `boolean`      | Shows expand toggle                     |
| `expanded`        | `boolean`      | Current expansion state                 |
| `onToggle`        | `() => void`   | Expand toggle callback                  |
| `onClick`         | `() => void`   | Row click callback                      |
| `selected`        | `boolean`      | Selection highlight                     |
| `viewStartOffset` | `number`       | Zoom window start (0-100)               |
| `viewEndOffset`   | `number`       | Zoom window end (0-100)                 |
| `orgName`         | `string`       | Organization name for SERVER label      |
| `children`        | `ReactNode`    | Nested content when expanded            |

### TimelineHeader

Combines time markers with the reusable `TimeBrush` component for timeline selection.

**Features:**

- Time markers at 0%, 25%, 50%, 75%, 100% with duration labels
- Uses the reusable `TimeBrush` component for selection
- Vertical guide lines for visual reference

```tsx
<TimelineHeader
  minTime={runStartTime}
  maxTime={runEndTime}
  leftWidth={40}
  onSelectionChange={(start, end) => setViewOffsets(start, end)}
/>
```

### TimeBrush (Reusable)

A standalone, reusable range selection component located at `src/TimeBrush/`.

**Features:**

- Draggable left/right handles to resize selection
- Drag selection area to move the window
- Click and drag on track to create new selection
- Hover cursor line indicates click position
- Reset button appears when selection differs from default
- Customizable styling via props

```tsx
import { TimeBrush } from '../TimeBrush';

<TimeBrush
  onSelectionChange={(start, end) => console.log(start, end)}
  initialStart={0}
  initialEnd={100}
  minSelectionWidth={2}
  selectionClassName="bg-primary-moderate/25"
  handleClassName="bg-primary-intense hover:bg-primary-xIntense"
>
  {/* Optional content to render inside the brush */}
  <div className="absolute inset-0 bg-blue-500" />
</TimeBrush>;
```

### Timeline (Container)

Orchestrates the full timeline rendering:

- Manages expansion state for all bars
- Converts `TimelineData` to `TimelineBar` props
- Handles zoom state from TimelineHeader
- Recursive rendering via `TimelineBarRenderer`

```tsx
<Timeline data={timelineData} onSelectStep={(stepId) => showStepDetails(stepId)} />
```

## Style System

Bar appearance is controlled by style keys that map to predefined configurations.
Colors use status-based semantics from the Status system for consistency (e.g., green for completed runs).

```typescript
type BarStyleKey =
  | 'root' // Root run bar (status color, checkbox icon)
  | 'step.run' // Standard step execution (status color)
  | 'step.sleep' // Fallback style (pending design)
  | 'step.waitForEvent' // Fallback style (pending design)
  | 'step.invoke' // Fallback style (pending design)
  | 'timing.inngest' // Queue time (short, gray)
  | 'timing.server' // Execution time (tall, barber-pole, status color)
  | 'timing.connecting' // Connection time (short, dotted border, status color)
  | 'default'; // Fallback style (gray)
```

Each style defines:

```typescript
interface BarStyle {
  barColor: string; // Tailwind background class (uses bg-status-* for status colors)
  textColor?: string; // Tailwind text class
  icon?: BarIcon; // Default icon
  pattern?: BarPattern; // 'solid' | 'barber-pole' | 'dotted'
  labelFormat?: string; // 'uppercase' | 'titlecase' | 'default'
  barHeight?: BarHeight; // 'short' | 'tall'
}
```

> **Note:** `step.sleep`, `step.waitForEvent`, and `step.invoke` currently use the default fallback
> style (gray) as their designs are pending from the design team.

## Compound Bars (Segments)

A single bar can contain multiple segments with different styles:

```typescript
const segments: BarSegment[] = [
  { id: 'queue', startPercent: 0, widthPercent: 30, style: 'timing.inngest' },
  { id: 'exec', startPercent: 30, widthPercent: 70, style: 'timing.server' },
];

<TimelineBar segments={segments} ... />
```

When expanded, compound bars show their segments as separate nested rows.

## Zooming (View Offsets)

The Time Brush selection drives zooming via `viewStartOffset` and `viewEndOffset`:

```
Full timeline:  |----------------------------------------|
                0%                                     100%

Zoomed view:              |--------------|
                         25%            75%
                          ↓              ↓
                viewStartOffset    viewEndOffset
```

Bars are clipped and scaled to fit the visible window:

- Bars fully outside the window are hidden
- Bars partially inside are clipped to window edges
- Positions are transformed to 0-100 within the visible window

## Data Types

### TimelineData

```typescript
interface TimelineData {
  minTime: Date; // Timeline start
  maxTime: Date; // Timeline end
  bars: TimelineBarData[]; // Root-level bars
  leftWidth: number; // Column divider position
  orgName?: string; // For "YOUR SERVER" label
}
```

### TimelineBarData

```typescript
interface TimelineBarData {
  id: string;
  name: string;
  startTime: Date;
  endTime: Date | null;
  style: BarStyleKey;
  children?: TimelineBarData[]; // Nested bars
  timingBreakdown?: {
    // For compound visualization
    queueMs: number;
    executionMs: number;
    totalMs: number;
  };
  isRoot?: boolean;
}
```

## Extending the System

### Adding a New Step Type

1. Add the style key to `BarStyleKey` in [TimelineBar.types.ts](./TimelineBar.types.ts)
2. Add the style configuration to `BAR_STYLES` in [TimelineBar.tsx](./TimelineBar.tsx)

```typescript
// types
type BarStyleKey = ... | 'step.newType';

// styles - use status colors for consistency
const BAR_STYLES = {
  'step.newType': {
    barColor: 'bg-status-completed', // Use status color for completed steps
    icon: 'gear',
  },
};
```

### Adding a New Icon

1. Import the icon from `@remixicon/react`
2. Add to `BarIcon` type and `ICON_MAP`

```typescript
import { RiNewIcon } from '@remixicon/react';

type BarIcon = ... | 'newIcon';

const ICON_MAP = {
  newIcon: RiNewIcon,
};
```

### Custom Nesting Logic

The recursive `TimelineBarRenderer` handles any nesting structure. To add new nesting patterns:

1. Add the parent/child relationship to your `TimelineBarData`
2. The renderer automatically handles expansion and indentation

## Constants

Key layout values in [utils/timing.ts](./utils/timing.ts):

```typescript
const TIMELINE_CONSTANTS = {
  MIN_BAR_WIDTH_PX: 2, // Minimum visible bar width
  INDENT_WIDTH_PX: 20, // Indentation per depth level
  BASE_LEFT_PADDING_PX: 4, // Base left padding
  ROW_HEIGHT_PX: 28, // Row height
  TRANSITION_MS: 150, // Animation duration
  DEFAULT_LEFT_WIDTH: 40, // Default column split
};
```

## File Structure

```
src/
├── TimeBrush/
│   ├── TimeBrush.tsx      # Reusable range selection component
│   └── index.ts           # Exports
└── RunDetailsV4/
    ├── Timeline.tsx           # Container component
    ├── TimelineBar.tsx        # Core bar component
    ├── TimelineBar.types.ts   # Type definitions
    ├── TimelineHeader.tsx     # Time markers + TimeBrush wrapper
    ├── utils/
    │   ├── formatting.ts      # Duration/label formatting
    │   └── timing.ts          # Position calculations, constants
    └── README.md              # This file
```
