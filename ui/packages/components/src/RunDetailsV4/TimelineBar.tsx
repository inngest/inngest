/**
 * TimelineBar component - Composable timeline row visualization.
 * Feature: 001-composable-timeline-bar
 *
 * This component renders a single row in the timeline with:
 * - Left panel: name, optional icon, duration
 * - Right panel: visual bar positioned by percentage
 * - Optional expansion to show nested children
 */

import { memo, useMemo, useState, type CSSProperties, type JSX } from 'react';
import {
  RiArrowRightFill,
  RiArrowRightLine,
  RiArrowRightSFill,
  RiBuilding2Line,
  RiCheckboxCircleFill,
  RiCloseCircleFill,
  RiFlashlightLine,
  RiFlaskLine,
  RiFunctionLine,
  RiInformationLine,
  RiMailLine,
  RiPercentLine,
  RiSettings3Line,
  RiStopCircleFill,
  RiTimeLine,
} from '@remixicon/react';
import { format } from 'date-fns';

import { formatVariantWeight } from '../Experiments/format';
import { HoverCardContent, HoverCardRoot, HoverCardTrigger } from '../HoverCard';
import { formatScoreValue } from '../RunDetails/ScoresAttrs';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { getStatusBackgroundClass, getStatusTextClass } from '../Status/statusClasses';
import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip/Tooltip';
import { cn } from '../utils/classNames';
import type {
  BarHeight,
  BarIcon,
  BarPattern,
  BarSegment,
  BarStyle,
  BarStyleKey,
  ScoreBadgeData,
  TimelineBarProps,
  TimingDetail,
} from './TimelineBar.types';
import { formatDuration } from './runDetailsUtils';
import { formatLabel } from './utils/formatting';
import { TIMELINE_CONSTANTS } from './utils/timing';

// ============================================================================
// Style Configurations
// ============================================================================

/**
 * Consolidated style configurations for all bar types.
 * Each entry contains visual style, status-based coloring flag, icon, height, and pattern.
 */
export const BAR_STYLES: Record<BarStyleKey, BarStyle> = {
  root: {
    barColor: 'bg-status-completed',
    barHeight: 'short',
    statusBased: true,
  },
  'step.run': {
    barColor: 'bg-status-completed',
    statusBased: true,
  },
  //
  // Waiting is the same substance as working with less of it, not a different
  // hue. A sleep or an unsatisfied wait is suspended, holding nothing and
  // costing nothing, so it is ghosted — same fill, a quarter of the weight —
  // while executing is solid. A hollow outline was tried first and read as a
  // border artefact rather than as a quantity.
  //
  // `step.invoke` keeps its full weight: a child run really is executing.
  'step.sleep': {
    barColor: 'bg-surfaceMuted',
    ghost: true,
    statusBased: false,
  },
  'step.waitForEvent': {
    barColor: 'bg-surfaceMuted',
    ghost: true,
    statusBased: false,
  },
  'step.invoke': {
    barColor: 'bg-status-completed',
    pattern: 'vertical-lines',
    statusBased: true,
  },
  'timing.waiting': {
    barColor: 'bg-status-completed',
    barHeight: 'tall',
    ghost: true,
    statusBased: true,
    labelFormat: 'default',
    textColor: 'text-light',
  },
  //
  // A backoff is DEAD TIME, and dead time already has an encoding.
  //
  // Colouring it by the attempt that follows painted a failure's consequence in
  // the colour of a success that had not happened yet. Colouring it by the
  // attempt that caused it overshot the other way: `retry`'s only user row went
  // 92% red on a run that succeeded, and the 6ms of green that makes it a
  // RECOVERY is about two pixels — so `retry` and `failure` read as the same
  // kind of event at a glance.
  //
  // The colour was being asked to carry both whose fault it was and how it
  // turned out, and it needs to carry neither. During a backoff the run is
  // suspended, holding nothing, costing nothing — exactly what `step.sleep` and
  // `step.waitForEvent` mean, and they are already neutral. Red attempt, neutral
  // gap, green attempt reads as "failed, waited, succeeded" without blaming an
  // interval in which nothing happened. The attempts carry the alarm, which is
  // why `failcluster` reads loudly: its MARKS are red, not its gaps.
  'timing.backoff': {
    barColor: 'bg-surfaceMuted',
    ghost: true,
    statusBased: false,
    labelFormat: 'default',
    textColor: 'text-light',
  },
  //
  // Time no span accounts for.
  //
  // Deliberately not called "processing", or latency, or queueing. It may be
  // any of those — what is actually known is that nothing reported it, and a
  // view whose value is that it does not invent should not start here. Drawn
  // hollow so it reads as an absence rather than as work.
  'timing.unaccounted': {
    // A colour, not a hollow outline. An outline reads as a border artefact
    // rather than as a quantity — the same reason waits stopped being hollow —
    // and this is real time the run spent, however little is known about it.
    // Accent rather than a status hue: it is neither success nor failure.
    barColor: 'bg-accent-subtle',
    statusBased: false,
    labelFormat: 'default',
    textColor: 'text-light',
  },
  'timing.inngest': {
    barColor: 'bg-surfaceMuted',
    barHeight: 'short',
    durationColor: 'text-basis',
    labelFormat: 'default',
    textColor: 'text-light',
  },
  //
  // The three things that can happen to a step before it runs, each its own
  // tone so a row reads left to right as a sequence rather than as one grey
  // stretch. All existing tokens — the palette is not extended for this.
  //
  //   queued       waiting its turn; the residual after the reported parts
  //   concurrency  held by a limit the user configured
  //   latency      Inngest's own overhead, reported per step
  'timing.inngest.queue': {
    barColor: 'bg-secondary-2xSubtle',
    labelFormat: 'default',
    textColor: 'text-light',
  },
  'timing.inngest.concurrency': {
    barColor: 'bg-secondary-moderate',
    labelFormat: 'default',
    textColor: 'text-light',
  },
  'timing.inngest.discovery': {
    // Solid, in a recessive tone. Discovery is the platform doing work, not the
    // run waiting, so it is not ghosted — and a hollow ring reads as a border
    // artefact rather than as a quantity.
    barColor: 'bg-surfaceMuted',
    barHeight: 'short',
    labelFormat: 'default',
    textColor: 'text-light',
  },
  'timing.inngest.finalization': {
    barColor: 'bg-secondary-xSubtle',
    labelFormat: 'default',
    textColor: 'text-light',
  },
  'timing.server': {
    barColor: 'bg-status-completed',
    barHeight: 'tall',
    durationColor: 'text-basis',
    labelFormat: 'default',
    pattern: 'barber-pole',
    statusBased: true,
    textColor: 'text-light',
  },
  'timing.connecting': {
    barColor: 'bg-status-completed',
    pattern: 'dotted',
    labelFormat: 'uppercase',
    barHeight: 'thin',
    statusBased: true,
  },
  // HTTP timing phases (children of SERVER bar)
  'timing.http.dns': {
    barColor: 'bg-status-completed',
    barHeight: 'thin',
    pattern: 'dotted',
    labelFormat: 'uppercase',
    statusBased: true,
  },
  'timing.http.tcp': {
    barColor: 'bg-status-completed',
    barHeight: 'thin',
    pattern: 'dotted',
    labelFormat: 'uppercase',
    statusBased: true,
  },
  'timing.http.tls': {
    barColor: 'bg-status-completed',
    barHeight: 'thin',
    pattern: 'dotted',
    labelFormat: 'uppercase',
    statusBased: true,
  },
  'timing.http.server': {
    barColor: 'bg-status-completed',
    barHeight: 'tall',
    pattern: 'barber-pole',
    labelFormat: 'uppercase',
    statusBased: true,
  },
  'timing.http.transfer': {
    barColor: 'bg-status-completed',
    barHeight: 'thin',
    pattern: 'dotted',
    labelFormat: 'uppercase',
    statusBased: true,
  },
  default: {
    barColor: 'bg-surfaceMuted',
    statusBased: true,
  },
};

/**
 * CSS pattern definitions for bar fills.
 * Barber-pole uses semi-transparent white stripes to work on any background color.
 */
export const BAR_PATTERNS: Record<BarPattern, CSSProperties> = {
  solid: {},
  'barber-pole': {
    backgroundImage: `repeating-linear-gradient(
      -45deg,
      transparent,
      transparent 6px,
      rgba(255, 255, 255, 0.15) 6px,
      rgba(255, 255, 255, 0.15) 8px
    )`,
  },
  'vertical-lines': {
    WebkitMaskImage: `repeating-linear-gradient(
      90deg,
      transparent,
      transparent 3px,
      black 3px,
      black 5px
    )`,
    maskImage: `repeating-linear-gradient(
      90deg,
      transparent,
      transparent 3px,
      black 3px,
      black 5px
    )`,
  },
  dotted: {
    WebkitMaskImage: `repeating-linear-gradient(
      90deg,
      black 0px,
      black 3px,
      transparent 3px,
      transparent 7px
    )`,
    maskImage: `repeating-linear-gradient(
      90deg,
      black 0px,
      black 3px,
      transparent 3px,
      transparent 7px
    )`,
  },
};

/**
 * Get the complete style configuration for a bar style key.
 * Falls back to 'default' if the key is not found.
 */
function getBarStyle(styleKey: BarStyleKey): BarStyle {
  return BAR_STYLES[styleKey];
}

/**
 * Get the bar color class, using status-based coloring when appropriate.
 */
function getBarColor(styleKey: BarStyleKey, status?: string): string {
  const barStyle = getBarStyle(styleKey);

  if (status && barStyle.statusBased) {
    return getStatusBackgroundClass(status);
  }

  return barStyle.barColor;
}

/**
 * Get the icon for a root bar based on run status.
 */
function getRootIcon(styleKey: BarStyleKey, status?: string): BarIcon | undefined {
  if (styleKey !== 'root') return undefined;
  switch (status) {
    case 'FAILED':
      return 'close-circle';
    case 'CANCELLED':
      return 'stop-circle';
    default:
      return 'checkbox';
  }
}

/**
 * Get the CSS pattern for a bar pattern type.
 */
function getBarPattern(pattern?: BarPattern): CSSProperties {
  return pattern ? BAR_PATTERNS[pattern] : BAR_PATTERNS.solid;
}

/**
 * Transform bar positions based on view offsets (for zooming).
 * Takes the original bar position and clips/scales it to the visible view window.
 *
 * @param startPercent - Original start position (0-100)
 * @param widthPercent - Original width (0-100)
 * @param viewStartOffset - Start of visible window (0-100, default 0)
 * @param viewEndOffset - End of visible window (0-100, default 100)
 * @returns Transformed start and width for the visible portion, or null if completely outside
 */
function transformBarPosition(
  startPercent: number,
  widthPercent: number,
  viewStartOffset: number = 0,
  viewEndOffset: number = 100
): { startPercent: number; widthPercent: number } | null {
  const barEnd = startPercent + widthPercent;
  const viewWidth = viewEndOffset - viewStartOffset;

  // If view width is 0 or negative, return null
  if (viewWidth <= 0) return null;

  // Check if bar is completely outside the view window
  if (barEnd <= viewStartOffset || startPercent >= viewEndOffset) {
    return null;
  }

  // Clip the bar to the view window
  const clippedStart = Math.max(startPercent, viewStartOffset);
  const clippedEnd = Math.min(barEnd, viewEndOffset);
  const clippedWidth = clippedEnd - clippedStart;

  // Transform to the 0-100 scale of the visible window
  const transformedStart = ((clippedStart - viewStartOffset) / viewWidth) * 100;
  const transformedWidth = (clippedWidth / viewWidth) * 100;

  return {
    startPercent: transformedStart,
    widthPercent: transformedWidth,
  };
}

// ============================================================================
// Icon Mapping
// ============================================================================

const ICON_MAP: Record<BarIcon, React.ComponentType<{ className?: string }>> = {
  gear: RiSettings3Line,
  building: RiBuilding2Line,
  lightning: RiFlashlightLine,
  function: RiFunctionLine,
  clock: RiTimeLine,
  mail: RiMailLine,
  arrow: RiArrowRightLine,
  checkbox: RiCheckboxCircleFill,
  'close-circle': RiCloseCircleFill,
  'stop-circle': RiStopCircleFill,
  experiment: RiFlaskLine,
  none: () => null,
};

/** Icons that should derive their color from run status */
const STATUS_ICONS = new Set<BarIcon>(['checkbox', 'close-circle', 'stop-circle']);

// ============================================================================
// Sub-components
// ============================================================================

/**
 * Renders the icon for a bar based on style or explicit prop.
 */
function BarIconComponent({
  icon,
  className,
  status,
}: {
  icon?: BarIcon;
  className?: string;
  status?: string;
}) {
  if (!icon || icon === 'none') return null;
  const IconComponent = ICON_MAP[icon];
  // Status icons (checkbox, close-circle, stop-circle) derive color from run status
  const statusColor = STATUS_ICONS.has(icon) && status ? getStatusTextClass(status) : undefined;
  return IconComponent ? (
    <IconComponent
      className={cn('h-3.5 w-3.5 shrink-0', className, statusColor)}
      data-testid="bar-icon"
    />
  ) : null;
}

/**
 * Renders the expand/collapse toggle button as a solid triangle.
 */
function ExpandToggle({
  expanded,
  onCollapse,
  selected,
  hoverCardOpen,
}: {
  expanded: boolean;
  onCollapse?: () => void;
  selected?: boolean;
  hoverCardOpen?: boolean;
}) {
  return (
    <div
      role={expanded ? 'button' : undefined}
      aria-label={expanded ? 'Collapse' : 'Expand'}
      className={cn(
        'flex items-center justify-center p-0',
        selected || hoverCardOpen ? 'bg-transparent' : 'bg-canvasBase',
        expanded && 'cursor-pointer'
      )}
      onClick={
        expanded
          ? (e) => {
              e.stopPropagation();
              onCollapse?.();
            }
          : undefined
      }
    >
      <RiArrowRightSFill
        className={cn('text-light h-4 w-4 transition-transform', expanded && 'rotate-90')}
        style={{ transitionDuration: `${TIMELINE_CONSTANTS.TRANSITION_MS}ms` }}
      />
    </div>
  );
}

/**
 * Hover card content for a timeline bar, showing duration/delay and start/end timestamps.
 */
function BarHoverCardContent({
  name,
  startTime,
  endTime,
  delayMs,
  timingDetails,
  styleLabel,
  segments,
}: {
  name: string;
  startTime: Date;
  endTime: Date | null;
  delayMs?: number;
  timingDetails?: TimingDetail[];
  styleLabel?: string;
  /** Every part of the bar under the pointer. More than one when they overlap. */
  segments?: BarSegment[] | null;
}) {
  // When the pointer is on a part of the bar, the card describes THAT PART.
  //
  // It used to lead with the segment's sentence and then report the row
  // underneath, which produced four separate contradictions in one card:
  // "Backed off 1.002s before attempt 2" above `DELAY -`; a 78ms discovery mark
  // reporting the Planning row's 774ms; every segment of a retried step
  // reporting the row's whole START/END; and `YOUR SERVER 1.015s` on a step
  // that spent 1.000s of it backing off. The last one passed the arithmetic
  // gate because it summed — the gate catches contradiction, not mislabelling.
  const stack = segments ?? [];
  const onSegment = stack.length > 0;
  // Timestamps only when the pointer is unambiguously on ONE thing. Two
  // concurrent requests under the cursor have two windows, and picking either
  // would be arbitrary; the lines below name both.
  const only = stack.length === 1 ? stack[0] : undefined;
  const segStart = only?.startMs !== undefined ? new Date(only.startMs) : null;
  const segEnd = only?.endMs !== undefined ? new Date(only.endMs) : null;

  // With SEVERAL things under the pointer there is no single window to report,
  // and falling back to the row's put its 774ms under two lines describing a
  // 180ms and a 151ms request — the contradiction this block was rewritten to
  // remove. The per-line durations above carry it instead.
  const ambiguous = stack.length > 1;
  const shownStart = segStart ?? startTime;
  const shownEnd = segStart ? segEnd : endTime;

  const startTimestamp = format(shownStart, 'yyyy-MM-dd HH:mm:ss.SSS');
  const endTimestamp = shownEnd ? format(shownEnd, 'yyyy-MM-dd HH:mm:ss.SSS') : null;

  const durationMs = shownEnd ? shownEnd.getTime() - shownStart.getTime() : 0;

  // A breakdown is shown only when it is one.
  //
  // These come from SDK metadata, and on several shapes they contradict the bar
  // they describe: `chains`' `left-2` reported DISCOVERY 229ms inside a 166ms
  // row — a component larger than the whole — and `retry`'s `first step`
  // credited YOUR SERVER with the full 1.093s when 1.003s of it was retry
  // backoff, giving INNGEST 98ms + YOUR SERVER 1.093s against a 1.093s total.
  //
  // Either the parts account for the whole or they are not a decomposition of
  // it, and presenting them as one is worse than leaving them out. The bar
  // itself is drawn from the timestamps, and the segment line below says what
  // the pointer is actually over.
  const detailSum = timingDetails?.reduce((n, d) => n + d.durationMs, 0) ?? 0;
  const detailsAddUp =
    durationMs > 0 &&
    detailSum <= durationMs * 1.02 + 1 &&
    (timingDetails ?? []).every((d) => d.durationMs <= durationMs);
  // Suppressed entirely while a segment is hovered: those figures decompose the
  // ROW, and the card is describing one part of it.
  const shownDetails = detailsAddUp && !onSegment ? timingDetails ?? [] : [];
  const hasDetails = shownDetails.length > 0;

  return (
    <div className="whitespace-nowrap px-1 py-0.5 text-xs">
      <p className="text-basis mb-1.5 font-medium">{name}</p>
      {styleLabel && <p className="text-light mb-1.5 font-mono text-[11px]">{styleLabel}</p>}

      {/* What the pointer is on. A row is several things — a wait, an attempt,
          a backoff, one member of a collapsed group — and the summary below
          describes all of them at once, which left a pale lead-in with no way
          of being asked what it was. */}
      {onSegment && (
        <div className="border-subtle mb-1.5 space-y-0.5 border-b pb-1.5">
          {stack.map(
            (part) =>
              part.tooltip && (
                <p key={part.id} className="text-basis">
                  {part.tooltip}
                  {stack.length > 1 &&
                    part.startMs !== undefined &&
                    part.endMs !== undefined &&
                    // Only where the label does not already say it. "Planned 2
                    // steps" needs the number; "Backed off 1.000s before attempt
                    // 2" was getting it twice.
                    !part.tooltip?.includes(formatDuration(part.endMs - part.startMs)) && (
                      <span className="text-light ml-1.5 tabular-nums">
                        {formatDuration(part.endMs - part.startMs)}
                      </span>
                    )}
                </p>
              )
          )}
        </div>
      )}
      <div className="flex flex-col gap-1">
        {/* Suppressed entirely when it would hold nothing. With a segment
            hovered the duration, delay and breakdown can all be gone at once,
            and the wrapper was still drawing its bottom rule — a horizontal
            line with nothing under it. */}
        {(!ambiguous || delayMs != null || hasDetails) && (
          <div
            className={cn(
              'flex flex-col gap-1',
              (hasDetails || delayMs != null) && 'border-subtle border-b pb-1.5'
            )}
          >
            {!ambiguous && (
              <div className="flex justify-between gap-6">
                <span className="text-light font-mono uppercase">Duration</span>
                <span className="text-basis tabular-nums">
                  {durationMs > 0 ? formatDuration(durationMs) : '-'}
                </span>
              </div>
            )}
            {delayMs != null && !onSegment && (
              <div className="flex justify-between gap-6">
                <span className="text-light font-mono uppercase">Delay</span>
                <span className="text-basis tabular-nums">
                  {delayMs > 0 ? formatDuration(delayMs) : '-'}
                </span>
              </div>
            )}
          </div>
        )}

        {hasDetails && (
          <div className="border-subtle flex flex-col gap-1 border-b pb-1.5">
            {shownDetails.map((detail) => (
              <div key={detail.label} className="flex justify-between gap-6">
                <span className="text-light font-mono uppercase">{detail.label}</span>
                <span className="text-basis tabular-nums">
                  {detail.durationMs > 0 ? formatDuration(detail.durationMs) : '-'}
                </span>
              </div>
            ))}
          </div>
        )}

        {!ambiguous && (
          <>
            <div
              className={cn(
                !hasDetails && delayMs == null && 'mt-0.5',
                'flex justify-between gap-6'
              )}
            >
              <span className="text-light font-mono uppercase">Start</span>
              <span className="text-basis tabular-nums">{startTimestamp}</span>
            </div>
            <div className="flex justify-between gap-6">
              <span className="text-light font-mono uppercase">End</span>
              {endTimestamp !== null ? (
                <span className="text-basis tabular-nums">{endTimestamp}</span>
              ) : (
                <span className="text-light italic">In progress</span>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

const BADGE_CLASS =
  'bg-canvasSubtle border-subtle text-subtle hover:bg-canvasMuted inline-flex h-5 w-5 items-center justify-center rounded border transition-colors';

/**
 * Experiment badge shown beside a step that ran under an experiment. Renders
 * the flask icon in the standard IconTile treatment (subtle bg + border, thin
 * rounded) with a hover card, and links to the experiment page when a
 * `pathCreator.experiment` is provided by the host app (cloud dashboard).
 */
function ExperimentBadge({ metadata }: { metadata?: TimelineBarProps['experimentMetadata'] }) {
  const { pathCreator } = usePathCreator();
  const href =
    pathCreator.experiment && metadata?.experimentName && metadata?.functionSlug
      ? pathCreator.experiment({
          experimentName: metadata.experimentName,
          functionSlug: metadata.functionSlug,
        })
      : null;

  const badge = <RiFlaskLine className="h-3 w-3" />;

  return (
    <HoverCardRoot openDelay={200} closeDelay={0}>
      <HoverCardTrigger asChild>
        {href ? (
          <a
            href={href}
            className={BADGE_CLASS}
            onClick={(e) => e.stopPropagation()}
            onMouseDown={(e) => e.stopPropagation()}
          >
            {badge}
          </a>
        ) : (
          <span
            className={BADGE_CLASS}
            onClick={(e) => e.stopPropagation()}
            onMouseDown={(e) => e.stopPropagation()}
          >
            {badge}
          </span>
        )}
      </HoverCardTrigger>
      <HoverCardContent side="top" align="start" className="border-muted max-w-none border">
        <ExperimentHoverCardContent metadata={metadata} />
      </HoverCardContent>
    </HoverCardRoot>
  );
}

/**
 * Hover card content for the experiment badge, showing experiment name and variant weights.
 * The selected variant is shown bold above the divider; other variants are shown lighter below.
 */
function ExperimentHoverCardContent({
  metadata,
}: {
  metadata?: TimelineBarProps['experimentMetadata'];
}) {
  if (!metadata) {
    return (
      <div className="whitespace-nowrap px-1 py-0.5 text-xs">
        <p className="text-light">No experiment data</p>
      </div>
    );
  }

  const { experimentName, variantSelected, variantWeights } = metadata;
  const availableVariants = metadata.availableVariants ?? [];

  // Separate selected variant from the rest
  const otherVariants = availableVariants.filter((v) => v !== variantSelected);

  return (
    <div className="whitespace-nowrap px-1 py-0.5 text-xs">
      <p className="text-light mb-0.5">Experiment name</p>
      <p className="text-basis mb-2 font-medium">{experimentName}</p>

      {(availableVariants.length > 0 || variantSelected) && (
        <div>
          {/* Header row */}
          <div className="border-subtle flex justify-between gap-6 border-b pb-1">
            <span className="text-light">Variant</span>
            {variantWeights && <span className="text-light">Weight</span>}
          </div>

          {/* Selected variant — bold, above the darker divider */}
          <div className="border-muted flex items-center justify-between gap-6 border-b py-1">
            <span className="text-basis flex items-center gap-1 font-medium">
              <RiArrowRightFill className="text-basis h-3 w-3 shrink-0" />
              {variantSelected}
            </span>
            {variantWeights && variantWeights[variantSelected] != null && (
              <span className="text-basis font-medium tabular-nums">
                {formatVariantWeight(variantWeights[variantSelected]!)}
              </span>
            )}
          </div>

          {/* Other variants — lighter */}
          {otherVariants.map((variant) => (
            <div
              key={variant}
              className="border-subtle flex items-center justify-between gap-6 border-b py-1 last:border-b-0"
            >
              <span className="text-light ml-4">{variant}</span>
              {variantWeights && variantWeights[variant] != null && (
                <span className="text-light tabular-nums">
                  {formatVariantWeight(variantWeights[variant]!)}
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * Score badge shown beside a span that recorded scores. Mirrors the experiment
 * badge treatment; scores have no dedicated page, so the badge never links out
 * and clicks fall through to row selection.
 */
function ScoreBadge({ scores }: { scores: ScoreBadgeData[] }) {
  return (
    <HoverCardRoot openDelay={200} closeDelay={0}>
      <HoverCardTrigger asChild>
        <span data-testid="score-badge" className={BADGE_CLASS}>
          <RiPercentLine className="h-3 w-3" />
        </span>
      </HoverCardTrigger>
      <HoverCardContent side="top" align="start" className="border-muted max-w-xs border">
        <ScoreHoverCardContent scores={scores} />
      </HoverCardContent>
    </HoverCardRoot>
  );
}

/**
 * Hover card content for the score badge, listing each recorded score name and
 * its formatted value.
 */
function ScoreHoverCardContent({ scores }: { scores: ScoreBadgeData[] }) {
  return (
    <div className="whitespace-nowrap px-1 py-0.5 text-xs">
      <div className="border-subtle flex justify-between gap-6 border-b pb-1">
        <span className="text-light">Score</span>
        <span className="text-light">Value</span>
      </div>
      {scores.map((score) => (
        <div
          key={score.name}
          className="border-subtle flex items-center justify-between gap-6 border-b py-1 last:border-b-0"
        >
          <span className="text-basis min-w-0 truncate font-medium" title={score.name}>
            {score.name}
          </span>
          <span className="text-basis shrink-0 font-mono tabular-nums">
            {formatScoreValue(score.value)}
          </span>
        </div>
      ))}
    </div>
  );
}

/**
 * Bars are thin, with a near-square radius rather than a pill. At an 18px row
 * a 16px-tall bar is the row; the bar should be a mark on the row, not fill it.
 */
/** Separation between adjacent segments within one bar. */
const SEGMENT_GAP_PX = 4;

/**
 * One height for everything.
 *
 * Three heights were meant to rank the kinds of bar, and what they actually did
 * was make a row of adjacent segments look ragged — the eye reads the step up
 * and down as well as along, and the variation carried no information the colour
 * and the label were not already carrying. Kept as a map so a style can still
 * ask for a height, but they all answer the same.
 */
const BAR_HEIGHT_CLASSES: Record<BarHeight, string> = {
  thin: 'h-3',
  short: 'h-3',
  tall: 'h-3',
};

/**
 * Renders the visual bar in the right panel.
 * For compound bars with segments, each segment is independently transformed
 * based on view offsets to only show the visible portion.
 */
const VisualBar = memo(function VisualBar({
  startPercent,
  widthPercent,
  style,
  segments,
  onSegmentHover,
  originalBarStart,
  originalBarWidth,
  viewStartOffset = 0,
  viewEndOffset = 100,
  status,
  expanded,
}: {
  startPercent: number;
  widthPercent: number;
  style: TimelineBarProps['style'];
  segments?: BarSegment[];
  /** Reports which segment the pointer is over, so the card can speak for it. */
  onSegmentHover?: (segments: BarSegment[] | null) => void;
  /** Original bar start before transform (for segment calculation) */
  originalBarStart?: number;
  /** Original bar width before transform (for segment calculation) */
  originalBarWidth?: number;
  /** View start offset for segment filtering */
  viewStartOffset?: number;
  /** View end offset for segment filtering */
  viewEndOffset?: number;
  /** Run status for status-based coloring */
  status?: string;
  /** Whether the parent row is expanded */
  expanded?: boolean;
}) {
  const barStyle = getBarStyle(style);
  const pattern = getBarPattern(barStyle.pattern);
  const heightClass = BAR_HEIGHT_CLASSES[barStyle.barHeight ?? 'tall'];
  const barColor = getBarColor(style, status);

  // Memoize segment transformation to avoid recalculating on every render
  const transformedSegments = useMemo(() => {
    if (!segments || segments.length === 0) return [];

    return (
      segments
        .map((segment) => {
          // Convert segment position from bar-relative to timeline-absolute
          const barStart = originalBarStart ?? 0;
          const barWidth = originalBarWidth ?? 100;
          const segmentAbsoluteStart = barStart + (segment.startPercent / 100) * barWidth;
          const segmentAbsoluteWidth = (segment.widthPercent / 100) * barWidth;

          // Transform to view coordinates
          const transformed = transformBarPosition(
            segmentAbsoluteStart,
            segmentAbsoluteWidth,
            viewStartOffset,
            viewEndOffset
          );

          if (!transformed) return null;

          return {
            ...segment,
            transformedStart: transformed.startPercent,
            transformedWidth: transformed.widthPercent,
          };
        })
        .filter(Boolean)
        // Widest first, so the NARROWEST end up last in the DOM and therefore on
        // top. Overlapping segments are common and the small ones were losing:
        // two of `chains`' five Planning marks sat almost entirely under a wider
        // neighbour, so hovering the 151ms request returned the 180ms one's card
        // and two of the five could not be pointed at at all. This moves nothing
        // and tells no lie about when anything happened — it only makes the
        // hard-to-hit ones the hittable ones.
        .sort((a, b) => (b?.transformedWidth ?? 0) - (a?.transformedWidth ?? 0))
    );
  }, [segments, originalBarStart, originalBarWidth, viewStartOffset, viewEndOffset]);

  // Render compound bar with segments if provided
  if (segments && segments.length > 0) {
    // If no segments are visible, don't render the container
    if (transformedSegments.length === 0) return null;

    return (
      <div
        data-testid="timeline-bar-visual"
        className="absolute h-full"
        style={{
          left: '0%',
          width: '100%',
          opacity: expanded ? 0 : 1,
        }}
      >
        {transformedSegments.map((segment) => {
          if (!segment) return null;
          const segmentStyle = getBarStyle(segment.style);
          const segmentPattern = getBarPattern(segmentStyle.pattern);
          const segmentHeightClass = BAR_HEIGHT_CLASSES[segmentStyle.barHeight ?? 'tall'];
          const segmentColor = getBarColor(segment.style, segment.status);
          const isOutlined = segmentStyle.outlined;
          // A ghosted segment is the same fill at a fraction of its weight, so
          // a step that queued then ran reads as one continuous bar in one
          // colour that simply gets more solid where the work is.
          const isGhost = segmentStyle.ghost;
          return (
            <div
              key={segment.id}
              // Each segment answers for itself, through the SAME hover card the
              // row uses. This shipped once as a native `title`, which was no
              // delivery at all: the row's card opens immediately on the same
              // pointer move, so the browser tooltip never won and the strings
              // never reached anyone. Two tooltips for one pointer is worse than
              // one, so there is only the card.
              onMouseEnter={() =>
                onSegmentHover?.(
                  // Everything under the pointer, not just the topmost.
                  //
                  // Concurrent requests overlap almost exactly — two of
                  // `chains`' five Planning marks sit on top of each other and
                  // 13 of `wide`'s stack in one band — so whichever loses the
                  // z-order was unreachable however the order was chosen. The
                  // overlap is a real fact about the run, so the card states it
                  // rather than the view trying to lay it out around.
                  transformedSegments.filter((other): other is NonNullable<typeof other> => {
                    if (!other) return false;
                    if (other.id === segment.id) return true;

                    // A MEANINGFUL overlap, not merely touching.
                    //
                    // Bare interval intersection collected neighbours that do
                    // not overlap on screen at all: adjacent segments share an
                    // exact boundary, and each bar subtracts SEGMENT_GAP_PX
                    // from its rendered width, so they sit 4px apart in pixels.
                    // That made `retry`'s four separate attempt segments return
                    // two stacked cards on a row measured as `overlaps=none`.
                    const overlap =
                      Math.min(
                        other.transformedStart + other.transformedWidth,
                        segment.transformedStart + segment.transformedWidth
                      ) - Math.max(other.transformedStart, segment.transformedStart);
                    const narrower = Math.min(other.transformedWidth, segment.transformedWidth);
                    return narrower > 0 && overlap > narrower * 0.25;
                  })
                )
              }
              onMouseLeave={() => onSegmentHover?.(null)}
              className={cn(
                'absolute top-1/2 -translate-y-1/2',
                segmentHeightClass,
                isOutlined ? 'bg-canvasBase' : segmentColor
              )}
              style={{
                left: `${segment.transformedStart}%`,
                // A hairline gap between adjacent segments. A row that goes
                // queued -> failed -> waited -> retried -> succeeded reads as
                // five things that happened rather than one striped bar, and
                // the gap is in pixels so it stays constant at any zoom.
                width: `calc(${segment.transformedWidth}% - ${SEGMENT_GAP_PX}px)`,
                minWidth: `${TIMELINE_CONSTANTS.MIN_BAR_WIDTH_PX}px`,
                ...(isOutlined
                  ? { boxShadow: 'inset 0 0 0 1px rgb(var(--color-background-surface-muted))' }
                  : isGhost
                  ? { opacity: 0.26 }
                  : segmentPattern),
              }}
            />
          );
        })}
      </div>
    );
  }

  // Render simple bar
  const isOutlined = barStyle.outlined;
  const isGhost = barStyle.ghost;
  return (
    <div
      data-testid="timeline-bar-visual"
      className={cn(
        'absolute top-1/2 -translate-y-1/2',
        heightClass,
        isOutlined ? 'bg-canvasBase' : barColor
      )}
      style={{
        left: `${startPercent}%`,
        width: `${widthPercent}%`,
        minWidth: `${TIMELINE_CONSTANTS.MIN_BAR_WIDTH_PX}px`,
        // A ghosted bar keeps its colour and loses its weight — waiting is the
        // same substance as work, just less of it.
        opacity: expanded ? 0 : isGhost ? 0.26 : 1,
        ...(isOutlined
          ? { boxShadow: 'inset 0 0 0 1px rgb(var(--color-background-surface-muted))' }
          : isGhost
          ? undefined
          : pattern),
      }}
    />
  );
});

// ============================================================================
// Main Component
// ============================================================================

/**
 * TimelineBar component renders a single row in the timeline visualization.
 *
 * Features:
 * - Configurable positioning via startPercent/widthPercent
 * - Style-based visual appearance
 * - Optional expand/collapse for nested children
 * - Optional icon display
 * - Depth-based indentation
 * - Selection highlighting
 */
export function TimelineBar({
  name,
  duration,
  icon,
  startPercent,
  widthPercent,
  depth,
  leftWidth,
  style,
  segments,
  expandable,
  expanded,
  onToggle,
  onClick,
  selected,
  hovered,
  onHoverChange,
  barID,
  note,
  interrupted,
  platform,
  badgeGutter = true,
  children,
  orgName,
  status,
  viewStartOffset = 0,
  viewEndOffset = 100,
  startTime,
  endTime,
  delayMs,
  actions,
  timingDetails,
  styleLabel,
  hasExperiment,
  insideExperiment,
  experimentMetadata,
  scores,
}: TimelineBarProps): JSX.Element {
  const showExperimentBackground = hasExperiment || insideExperiment;
  const barStyle = getBarStyle(style);
  const effectiveIcon = icon ?? barStyle.icon ?? getRootIcon(style, status);

  // Format the display name based on style
  let displayName = name;
  if (barStyle.labelFormat) {
    displayName = formatLabel(name, barStyle.labelFormat);
  }

  // For SERVER timing, show org name or fallback
  if (style === 'timing.server') {
    displayName = orgName ? `${orgName} server` : 'Your server';
  }

  // Calculate indentation (base padding + depth-based indent)
  const indentPx =
    TIMELINE_CONSTANTS.BASE_LEFT_PADDING_PX + depth * TIMELINE_CONSTANTS.INDENT_WIDTH_PX;

  // Transform bar position based on view offsets
  const transformed = useMemo(
    () => transformBarPosition(startPercent, widthPercent, viewStartOffset, viewEndOffset),
    [startPercent, widthPercent, viewStartOffset, viewEndOffset]
  );

  // Hover card state — controlled so hover target (full right panel) is separate from anchor (bar position)
  const showHoverCard = !!startTime;
  const [hoverCardOpen, setHoverCardOpen] = useState(false);
  // Which part of the bar the pointer is over, so the card answers for that
  // part rather than returning the row's summary for all of it.
  const [hoveredSegment, setHoveredSegment] = useState<BarSegment[] | null>(null);

  return (
    <div data-testid="timeline-bar-container" className="relative">
      {/* Main row */}
      <div
        data-testid="timeline-bar-row"
        // So the flow overlay can find this row's position after layout,
        // whatever is expanded above it.
        data-bar-id={barID}
        className="relative isolate flex cursor-pointer items-center"
        onMouseEnter={() => onHoverChange?.(true)}
        onMouseLeave={() => onHoverChange?.(false)}
        onClick={() => {
          onClick?.();
          if (expandable) {
            onToggle?.();
          }
        }}
        style={{ height: `${TIMELINE_CONSTANTS.ROW_HEIGHT_PX}px` }}
      >
        {/* Selection / hover highlight - extends from indent to full width */}
        {(selected || hovered || hoverCardOpen) && (
          <div
            className={cn(
              'pointer-events-none absolute inset-y-0 right-0 -z-10',
              selected ? 'bg-secondary-3xSubtle' : 'bg-canvasSubtle'
            )}
            style={{
              left: `${indentPx - 24}px`,
            }}
          />
        )}
        {/* Left panel - name, icon, controls */}
        <div
          data-testid="timeline-bar-left"
          className="relative flex h-full shrink-0 items-center gap-1.5 overflow-hidden pr-2"
          style={{
            width: `${leftWidth}%`,
            paddingLeft: `${indentPx}px`,
          }}
        >
          {/* Expand toggle - absolutely positioned to sit on the parent's vertical line */}
          {expandable && (
            <div
              className="absolute flex items-center justify-center"
              style={{ left: `${indentPx - 20}px` }}
            >
              <ExpandToggle
                expanded={expanded ?? false}
                onCollapse={onToggle}
                selected={selected}
                hoverCardOpen={hoverCardOpen}
              />
            </div>
          )}

          {/* Icon */}
          <BarIconComponent icon={effectiveIcon} className="text-subtle ml-px" status={status} />

          {/* Name + actions wrapper */}
          <div className="flex min-w-0 flex-1 items-center">
            {/* Name */}
            <span
              className={cn(
                // The label column should recede — the bars are the content.
                // Monospace, muted, truncating, and never bold.
                // A row's NAME outranks its annotation.
                //
                // The note was `shrink-0` while this was `flex-1 min-w-0`, so
                // the annotation never yielded and the identity absorbed all of
                // the loss: `blocked`'s Run row rendered its name at ONE PIXEL —
                // not even the ellipsis fit — leaving the row that anchors the
                // whole timeline identifiable only by its icon. The floor here
                // is what the note now shrinks against.
                // `flex-auto`, not `flex-1`. `flex-1` sets `flex-basis: 0`, so
                // the name has no base to shrink FROM and the note takes its
                // content width first whatever its shrink factor — which is how
                // the note came to win outright. With a content base, the
                // shrink factors below actually apply.
                'min-w-0 flex-auto overflow-hidden text-ellipsis whitespace-nowrap font-mono text-[11px] font-normal leading-tight',
                // The platform's own rows recede further than the user's.
                //
                // Finalization and Function error are truthful and earn their
                // place, but on `failure` the widest bar after Run is 272ms of
                // Function error while the thing the reader came to find is a
                // 43ms step above it. The bar keeps its status colour, so a
                // failure is still findable; it is the NAME that stops
                // competing with names the reader actually wrote.
                platform ? 'text-light' : barStyle.textColor ?? 'text-subtle',
                !effectiveIcon && 'pl-1.5'
              )}
            >
              {displayName}
              {(style === 'timing.inngest' || style === 'timing.server') && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span
                      className=" ml-1 inline-flex shrink-0 cursor-help align-middle"
                      onClick={(e) => e.stopPropagation()}
                      onMouseDown={(e) => e.stopPropagation()}
                    >
                      <RiInformationLine className="text-light h-3.5 w-3.5" />
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="top" className=" max-w-xs text-xs shadow-lg">
                    {style === 'timing.inngest'
                      ? 'Time spent on queue delays, concurrency limits, processing delays, and related overhead'
                      : 'Time spent on your server executing the function'}
                  </TooltipContent>
                </Tooltip>
              )}
            </span>

            {/* A faded aside — the run's queue delay, when it was too small to
                be worth spending plot width on. */}
            {note && (
              <span
                // Shrinks to nothing before the name loses a character.
                //
                // It was `shrink-0`, so the annotation never yielded and the
                // identity absorbed all the loss: `blocked`'s Run row rendered
                // its name at ONE PIXEL — not even the ellipsis fit — and
                // `loop40`'s `act-[0…39] × 40` lost the `× 40` that says it is a
                // group while its neighbour kept it. An enormous shrink factor
                // states the priority directly, where a minimum width on the
                // name only moved the threshold.
                // Shrinks hard, but not below legibility. `+…` says nothing and
                // looks like a glitch, and on `v4pathological` the note reached
                // ZERO pixels — so a row read `254ms` against a canvas node
                // reading `1ms` with the number that reconciles them absent.
                // The reconciliation is only as good as the note's ability to
                // render.
                //
                // Below this floor the NAME takes the truncation instead, which
                // is the better trade: the name's ellipsis works and carries
                // meaning, and the note survives in full in the hover card.
                className="text-light ml-1.5 min-w-[3rem] shrink-[9999] overflow-hidden text-ellipsis whitespace-nowrap font-mono text-[10px] tabular-nums"
              >
                {note}
              </span>
            )}

            {/* Actions slot */}
            {actions}
          </div>

          {/* Duration. Tabular numerals so the column of digits aligns down the
              run — the comparison between rows is the whole point of it. */}
          <span
            className={cn(
              'shrink-0 font-mono text-[11px] tabular-nums',
              barStyle.durationColor ?? barStyle.textColor ?? 'text-muted'
            )}
          >
            {formatDuration(duration)}
            {/* A floor, not a measurement: this step never finished, so the
                number says how long it had run when the run ended. Naming that
                beats a bare number the reader would take as a duration. */}
            {interrupted && <span className="text-light ml-1">cut short</span>}
          </span>
        </div>

        {/* Experiment and score badges - centered between left panel and bars.
            Width is fixed so bars stay aligned across rows regardless of badges. */}
        {/* Fixed-width so bars stay aligned across rows regardless of badges —
            but only when some row in this timeline actually has one. It was
            48px of dead space on every fixture otherwise. */}
        <span
          className={cn(
            'inline-flex shrink-0 items-center justify-center gap-1',
            badgeGutter ? 'w-12' : 'w-1'
          )}
        >
          {hasExperiment && <ExperimentBadge metadata={experimentMetadata} />}
          {scores && scores.length > 0 && <ScoreBadge scores={scores} />}
        </span>

        {/* Right panel - visual bar with optional hover card */}
        <div
          data-testid="timeline-bar-right"
          className="relative h-full flex-1"
          style={{ width: `${100 - leftWidth}%` }}
          onMouseEnter={showHoverCard ? () => setHoverCardOpen(true) : undefined}
          onMouseLeave={
            showHoverCard
              ? () => {
                  setHoverCardOpen(false);
                  setHoveredSegment(null);
                }
              : undefined
          }
        >
          {/* No centre line, and no grid. One structural rule only — the
              vertical divider between the label column and the plot area, drawn
              once by the container rather than once per row. Every horizontal
              rule here was a divider that separated nothing. */}

          {/* Dotted background pattern for experiment steps and their children */}
          {showExperimentBackground && (
            <div
              className="bg-canvasSubtle pointer-events-none absolute inset-0"
              style={{
                backgroundImage:
                  'radial-gradient(circle, rgb(var(--color-border-muted)) 1px, transparent 1px)',
                backgroundSize: '7px 7px',
              }}
            />
          )}

          {/* Bar container, centered vertically */}
          <div className="absolute inset-y-0 flex w-full items-center">
            {transformed && (
              <>
                <VisualBar
                  startPercent={transformed.startPercent}
                  widthPercent={transformed.widthPercent}
                  style={style}
                  segments={segments}
                  onSegmentHover={setHoveredSegment}
                  originalBarStart={startPercent}
                  originalBarWidth={widthPercent}
                  viewStartOffset={viewStartOffset}
                  viewEndOffset={viewEndOffset}
                  status={status}
                  expanded={expandable && expanded}
                />
                {showHoverCard && (
                  <HoverCardRoot open={hoverCardOpen} closeDelay={0}>
                    <HoverCardTrigger asChild>
                      <div
                        className="pointer-events-none absolute inset-y-0"
                        style={{
                          left: `${transformed.startPercent}%`,
                          width: `${transformed.widthPercent}%`,
                          minWidth: '4px',
                        }}
                      />
                    </HoverCardTrigger>
                    <HoverCardContent side="top" className="border-muted max-w-none border">
                      <BarHoverCardContent
                        name={displayName}
                        startTime={startTime!}
                        endTime={endTime ?? null}
                        delayMs={delayMs}
                        timingDetails={timingDetails}
                        styleLabel={styleLabel}
                        segments={hoveredSegment}
                      />
                    </HoverCardContent>
                  </HoverCardRoot>
                )}
              </>
            )}
          </div>
        </div>
      </div>

      {/* Vertical guide line from arrow to bottom of expanded area */}
      {expanded && (
        <div
          className="bg-canvasMuted absolute w-px"
          style={{
            left: `${indentPx + 8}px`,
            top: `${TIMELINE_CONSTANTS.ROW_HEIGHT_PX}px`,
            bottom: 0,
          }}
        />
      )}

      {/* Children (expanded content) */}
      {expanded && children}
    </div>
  );
}
