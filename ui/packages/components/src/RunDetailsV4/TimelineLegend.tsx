/**
 * TimelineLegend - Hover popover showing all timeline bar styles.
 * Displayed next to the "Trace" tab heading.
 *
 * Derives all visual styles from BAR_STYLES and BAR_PATTERNS in TimelineBar.tsx
 * so the legend automatically stays in sync when bar styles change.
 */

import { RiInformationLine } from '@remixicon/react';

import { HoverCardContent, HoverCardRoot, HoverCardTrigger } from '../HoverCard';
import { getStatusBackgroundClass } from '../Status/statusClasses';
import { BAR_PATTERNS, BAR_STYLES } from './TimelineBar';
import type { BarStyleKey } from './TimelineBar.types';

// ============================================================================
// Bar sample rendering
// ============================================================================

const SAMPLE_WIDTH = 32;
const SAMPLE_HEIGHTS: Record<string, number> = { thin: 2, short: 12, tall: 20 };

type LegendEntry = {
  label: string;
  style: BarStyleKey;
};

function BarSample({ label, style }: LegendEntry) {
  const barStyle = BAR_STYLES[style] ?? BAR_STYLES.default;
  const height = SAMPLE_HEIGHTS[barStyle.barHeight ?? 'tall'] ?? SAMPLE_HEIGHTS.tall;
  const isOutlined = barStyle.outlined;
  const colorClass = isOutlined
    ? 'bg-canvasBase'
    : barStyle.statusBased
    ? getStatusBackgroundClass('COMPLETED')
    : barStyle.barColor;
  const patternCss = isOutlined
    ? { boxShadow: 'inset 0 0 0 1px rgb(var(--color-background-surface-muted))' }
    : barStyle.pattern
    ? BAR_PATTERNS[barStyle.pattern] ?? {}
    : {};

  return (
    <div className="flex items-center gap-2">
      <div
        className={`${colorClass} shrink-0 rounded-sm`}
        style={{
          width: SAMPLE_WIDTH,
          height,
          ...patternCss,
        }}
      />
      <span className="text-muted whitespace-nowrap text-xs">{label}</span>
    </div>
  );
}

// ============================================================================
// Legend groups — each entry references a BarStyleKey
// ============================================================================

const functionBars: LegendEntry[] = [
  { label: 'step.run', style: 'step.run' },
  { label: 'step.sleep / waitForEvent / invoke', style: 'step.sleep' },
  { label: 'Run', style: 'root' },
];

const inngestBars: LegendEntry[] = [
  { label: 'Planned delay (queue, concurrency)', style: 'timing.inngest.queue' },
  { label: 'System latency (discovery, finalization)', style: 'timing.inngest.discovery' },
];

const serverBars: LegendEntry[] = [
  { label: 'Execution time', style: 'timing.server' },
  { label: 'DNS lookup', style: 'timing.http.dns' },
  { label: 'TCP connection', style: 'timing.http.tcp' },
  { label: 'TLS handshake', style: 'timing.http.tls' },
  { label: 'Server processing', style: 'timing.http.server' },
  { label: 'Content transfer', style: 'timing.http.transfer' },
];

function LegendGroup({ title, items }: { title: string; items: LegendEntry[] }) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-basis text-xs font-medium">{title}</span>
      {items.map((item) => (
        <BarSample key={item.label} {...item} />
      ))}
    </div>
  );
}

// ============================================================================
// Export
// ============================================================================

export function TimelineLegend() {
  return (
    <HoverCardRoot openDelay={200} closeDelay={100}>
      <HoverCardTrigger asChild>
        <span className="ml-1 inline-flex shrink-0 cursor-help align-middle">
          <RiInformationLine className="text-light h-3.5 w-3.5" />
        </span>
      </HoverCardTrigger>
      <HoverCardContent side="bottom" align="start" className="w-auto max-w-sm">
        <div className="flex flex-col gap-3 p-1">
          <LegendGroup title="Function" items={functionBars} />
          <LegendGroup title="Inngest" items={inngestBars} />
          <LegendGroup title="Your server" items={serverBars} />
        </div>
      </HoverCardContent>
    </HoverCardRoot>
  );
}
