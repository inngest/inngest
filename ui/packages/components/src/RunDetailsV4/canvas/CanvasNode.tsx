/**
 * Node renderers for the Run Canvas.
 *
 * The point of this view is that a user recognises their own code in it, so
 * every node names the SDK call that produced it (`step.run`, `step.waitForEvent`)
 * in monospace, and each step type carries a distinct icon. Status colours come
 * from the same `Status/*` helpers the timeline uses.
 *
 * Colour contract (a distinction the trace does not currently draw):
 *   green = success · red = failure · grey = timed out or not yet run.
 * A timeout is neither a success nor a bug in the step, so it gets its own
 * neutral treatment — including invoke timeouts, not just waits.
 */
import { useState } from 'react';
import {
  RiCheckboxCircleFill,
  RiCloseCircleFill,
  RiExpandDiagonalLine,
  RiFlashlightLine,
  RiFunctionLine,
  RiLoader4Line,
  RiMailLine,
  RiMoonLine,
  RiRadarLine,
  RiShareForwardBoxLine,
  RiSignalTowerLine,
  RiSparkling2Line,
  RiStopCircleFill,
  RiTimeLine,
} from '@remixicon/react';
import { Handle, Position, type Node, type NodeProps } from '@xyflow/react';

import { getStatusBorderClass, getStatusTextClass } from '../../Status/statusClasses';
import { cn } from '../../utils/classNames';
import { formatDuration } from '../runDetailsUtils';
import { groupTitle, type CanvasGroup, type CollapseException } from './collapse';
import { LAYOUT, type CanvasNodeData } from './toFlowElements';

/** One icon per step type, so the shape of a run is readable at a glance. */
const STEP_TYPE_ICON = {
  RUN: RiFunctionLine,
  SLEEP: RiMoonLine,
  WAIT_FOR_EVENT: RiMailLine,
  WAIT_FOR_SIGNAL: RiSignalTowerLine,
  INVOKE: RiShareForwardBoxLine,
  AI_GATEWAY: RiSparkling2Line,
} as const;

/** The SDK call the user actually wrote. */
const STEP_TYPE_LABEL = {
  RUN: 'step.run',
  SLEEP: 'step.sleep',
  WAIT_FOR_EVENT: 'step.waitForEvent',
  WAIT_FOR_SIGNAL: 'step.waitForSignal',
  INVOKE: 'step.invoke',
  AI_GATEWAY: 'step.ai.infer',
} as const;

const STATUS_ICON = {
  COMPLETED: RiCheckboxCircleFill,
  FAILED: RiCloseCircleFill,
  CANCELLED: RiStopCircleFill,
  RUNNING: RiLoader4Line,
  QUEUED: RiTimeLine,
  WAITING: RiTimeLine,
  SKIPPED: RiStopCircleFill,
  UNKNOWN: RiTimeLine,
} as const;

function stepTypeKey(data: CanvasNodeData) {
  const op = (data.stepOp ?? '').toUpperCase();
  return op in STEP_TYPE_ICON ? (op as keyof typeof STEP_TYPE_ICON) : null;
}

function iconFor(data: CanvasNodeData) {
  if (data.kind === 'event') return RiFlashlightLine;
  if (data.kind === 'result') return STATUS_ICON[data.status];
  const key = stepTypeKey(data);
  return key ? STEP_TYPE_ICON[key] : RiFunctionLine;
}

function durationOf(data: CanvasNodeData): string | null {
  if (data.endedAt === null) return null;
  return formatDuration(data.endedAt - (data.startedAt ?? data.queuedAt));
}

const HANDLE = '!h-1 !w-1 !min-w-0 !border-0 !bg-transparent';

/**
 * Neutral, so selection never competes with a status colour — a selected failed
 * step should not show a red stroke and a green one at once.
 *
 * The tailwind preset extends borderColor but not ringColor, so `ring-contrast`
 * is not a real class and silently falls back to Tailwind's default blue. Read
 * the token directly instead.
 */
const SELECTED_RING =
  'ring-2 ring-offset-2 ring-[rgb(var(--color-border-contrast))] ring-offset-[rgb(var(--color-background-canvas-subtle))]';

/** Grey. Used for timeouts and for work that has not run yet. */
const NEUTRAL = { border: 'border-muted border', text: 'text-muted' };

export function CanvasStepNode({ data, selected }: NodeProps<Node<CanvasNodeData>>) {
  const Icon = iconFor(data);
  const duration = durationOf(data);
  const isEvent = data.kind === 'event';
  const isResult = data.kind === 'result';
  const timedOut = data.waitTimedOut === true;
  const typeKey = stepTypeKey(data);

  const filled = isResult && (data.status === 'COMPLETED' || data.status === 'FAILED');
  const onFill = filled ? 'text-alwaysWhite' : undefined;

  const shell = isEvent
    ? 'bg-canvasBase border-subtle border'
    : isResult
    ? cn(
        'border',
        getStatusBorderClass(data.status),
        data.status === 'COMPLETED'
          ? 'bg-status-completed'
          : data.status === 'FAILED'
          ? 'bg-status-failed'
          : 'bg-canvasBase'
      )
    : cn(
        'bg-canvasBase',
        // A timeout is not a failure of the step, so it goes grey rather than red.
        timedOut ? NEUTRAL.border : getStatusBorderClass(data.status)
      );

  const accent = filled
    ? 'text-alwaysWhite'
    : isEvent
    ? 'text-subtle'
    : timedOut
    ? NEUTRAL.text
    : getStatusTextClass(data.status);

  return (
    <div className="relative" style={{ width: LAYOUT.nodeWidth, height: LAYOUT.nodeHeight }}>
      {/* A batch is drawn as a stack of cards rather than as N separate nodes. */}
      {isEvent && (data.batchSize ?? 0) > 1 && (
        <>
          <div className="border-subtle bg-canvasBase absolute inset-0 translate-x-1.5 translate-y-1.5 rounded-full border" />
          <div className="border-subtle bg-canvasBase absolute inset-0 translate-x-[3px] translate-y-[3px] rounded-full border" />
        </>
      )}

      <div
        className={cn(
          'absolute inset-0 flex flex-col justify-center gap-0.5 px-3 py-1.5 transition-all',
          isEvent || isResult ? 'rounded-full' : 'rounded-md',
          // Waits are dashed, echoing the timeline's vertical-lines pattern for
          // sleep / waitForEvent / invoke bars.
          data.kind === 'wait' && 'border-dashed',
          shell,
          selected && SELECTED_RING,
          data.future && 'opacity-25'
        )}
      >
        <Handle type="target" position={Position.Left} className={HANDLE} />

        <div className="flex items-center gap-1.5">
          <Icon
            className={cn(
              'h-3.5 w-3.5 shrink-0',
              accent,
              data.status === 'RUNNING' && !isEvent && 'animate-spin'
            )}
          />
          <span
            className={cn('truncate text-xs font-medium', onFill ?? 'text-basis')}
            title={data.duplicateIndex ? `${data.label} (#${data.duplicateIndex})` : data.label}
          >
            {data.label}
          </span>
          {/* The SDK's own collision index, so two steps sharing a name are
              still distinguishable. */}
          {data.duplicateIndex ? (
            <span className="text-muted shrink-0 font-mono text-[10px] leading-none">
              #{data.duplicateIndex}
            </span>
          ) : null}
        </div>

        <div className="flex items-center gap-1.5 overflow-hidden text-[10px] leading-none">
          {isEvent ? (
            <span className={cn('truncate font-mono', NEUTRAL.text)}>
              {(data.batchSize ?? 0) > 1 ? `${data.batchSize} events` : 'event'}
            </span>
          ) : isResult ? (
            duration && <span className={cn('font-mono', onFill ?? 'text-subtle')}>{duration}</span>
          ) : (
            <>
              {typeKey && (
                <span className="text-muted shrink-0 truncate font-mono">
                  {STEP_TYPE_LABEL[typeKey]}
                </span>
              )}
              {timedOut ? (
                <span className={cn('shrink-0', NEUTRAL.text)}>timed out</span>
              ) : (
                duration && <span className="text-subtle shrink-0 font-mono">{duration}</span>
              )}
              {data.attempts > 0 && (
                <span className="text-status-failedText shrink-0">{data.attempts + 1}×</span>
              )}
            </>
          )}
        </div>

        <Handle type="source" position={Position.Right} className={HANDLE} />
      </div>
    </div>
  );
}

/**
 * Where the run changes shape: one step becoming several, or several converging
 * back into one. Small, because it is a junction rather than work.
 *
 * It is also a real request — the executor asked the function what to do next —
 * so clicking it says when that happened and what came back. Rendered as a
 * popover rather than in the step panel because a discovery is not a step: it
 * has no input, output or attempts to show there.
 */
export function CanvasJoinNode({ data, selected }: NodeProps<Node<CanvasNodeData>>) {
  const [open, setOpen] = useState(false);
  const discovery = data.discovery;
  const took =
    discovery && discovery.endedAt !== null && discovery.startedAt !== null
      ? formatDuration(discovery.endedAt - discovery.startedAt)
      : null;

  const shape = data.junction;

  // The narrowing case is the one worth naming. The executor does not ask the
  // function what to do next as each parallel branch finishes: it skips every
  // one but the last, and dedupes branches that finish together on a shared
  // key. N completions become a single request.
  const headline =
    shape?.direction === 'converge'
      ? 'Back to sequential'
      : shape?.direction === 'both'
      ? 'Regrouped'
      : 'Running in parallel';

  const steps = (count: number) => `${count} step${count === 1 ? '' : 's'}`;

  const explanation =
    shape?.direction === 'converge'
      ? shape.unproven > 0
        ? // Some of what arrives here only *might* have been waited for, so
          // this says what is certain — the ordering — and leaves the broken
          // lines to carry the rest.
          `${steps(shape.from)} finished here, and the run carried on as one.`
        : `All ${steps(
            shape.from
          )} finished, so Inngest carried on with a single request rather than ${shape.from}.`
      : shape?.direction === 'both'
      ? `${steps(shape.from)} finished, and Inngest started ${steps(shape.to)} in parallel.`
      : `Inngest started ${steps(shape?.to ?? 0)} in parallel.`;

  return (
    <div className="relative" style={{ width: LAYOUT.joinWidth, height: LAYOUT.joinWidth }}>
      <button
        type="button"
        disabled={!shape}
        onClick={(event) => {
          event.stopPropagation();
          setOpen((o) => !o);
        }}
        className={cn(
          'bg-canvasBase text-muted flex h-full w-full items-center justify-center rounded-full border transition-all',
          data.status === 'FAILED' ? getStatusBorderClass('FAILED') : 'border-muted',
          (selected || open) && SELECTED_RING,
          shape && 'hover:text-basis cursor-pointer',
          data.future && 'opacity-25'
        )}
        title={shape ? headline : 'Where the run splits or rejoins'}
      >
        <Handle type="target" position={Position.Left} className={HANDLE} />
        <RiRadarLine className="h-3 w-3" />
        <Handle type="source" position={Position.Right} className={HANDLE} />
      </button>

      {open && (
        <div
          className="nodrag nopan border-subtle bg-canvasBase absolute left-1/2 top-full z-10 mt-1.5 w-52 -translate-x-1/2 rounded border p-2 text-[10px] shadow-sm"
          onClick={(event) => event.stopPropagation()}
        >
          <div className="text-basis mb-1 flex items-baseline justify-between gap-2 leading-none">
            <span className="font-medium">{headline}</span>
            {took && <span className="text-muted font-mono">{took}</span>}
          </div>

          <p className="text-muted leading-snug">{explanation}</p>
        </div>
      )}
    </div>
  );
}

/**
 * A repeated shape drawn once.
 *
 * The hard part is not the count, it is not throwing away what differed. A
 * 40-iteration agent loop where iteration 7 called a different tool is exactly
 * the run someone opened the trace to understand, so this node carries:
 *
 *  - the variance, as "38 x search, 2 x write_file" when names actually differ,
 *    and as a `first … last` range when they are only numbered;
 *  - the exceptions, naming how many failed, retried or ran long;
 *  - a sparkline of every member's duration, in run order, so a slowdown partway
 *    through is visible without expanding anything;
 *  - the group's worst status as its border, so a group holding a failure is
 *    never drawn as a green one.
 *
 * And it expands. A collapsed thing you cannot open is worse than no collapsing.
 */
function Sparkline({ values, className }: { values: number[]; className?: string }) {
  if (values.length < 2) return null;

  const max = Math.max(...values);
  if (max <= 0) return null;

  // One point per member, even at 500 — the shape of the run over its iterations
  // is the signal, and thinning it would flatten exactly the spike worth seeing.
  const step = 100 / (values.length - 1);
  const points = values.map((v, i) => `${i * step},${10 - (v / max) * 9}`).join(' ');

  return (
    <svg
      className={cn('h-2.5 w-full', className)}
      viewBox="0 0 100 10"
      preserveAspectRatio="none"
      aria-hidden
    >
      <polyline
        points={points}
        fill="none"
        stroke="currentColor"
        strokeWidth={0.8}
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}

/** "2 failed, 1 slow" — plural-correct, most serious first. */
function summariseExceptions(group: CanvasGroup): string | null {
  const order: Array<[CollapseException['reason'], string]> = [
    ['failed', 'failed'],
    ['cancelled', 'cancelled'],
    ['running', 'still running'],
    ['retried', 'retried'],
    ['slow', 'slow'],
  ];
  const counts = new Map<string, number>();
  for (const e of group.exceptions) counts.set(e.reason, (counts.get(e.reason) ?? 0) + 1);

  const parts = order
    .filter(([reason]) => counts.has(reason))
    .map(([reason, label]) => `${counts.get(reason)} ${label}`);

  return parts.length ? parts.join(', ') : null;
}

export function CanvasGroupNode({ data, selected }: NodeProps<Node<CanvasNodeData>>) {
  const group = data.group;
  if (!group) return null;

  const Icon = iconFor(data);
  const typeKey = stepTypeKey(data);
  const failing = group.status === 'FAILED';

  // Numbered names carry no information beyond the index, so they read as a
  // range. A genuine mix of names is the interesting case and gets the counts.
  const title = groupTitle(group);

  const variantLine =
    group.variance === 'mixed'
      ? group.variants
          .slice(0, 3)
          .map((v) => `${v.count} x ${v.label}`)
          .join(', ')
      : null;

  const exceptions = summariseExceptions(group);
  const elapsed = formatDuration(group.envelope.lastEndedAt - group.envelope.firstStartedAt);

  return (
    <div className="relative" style={{ width: LAYOUT.nodeWidth, height: LAYOUT.groupHeight }}>
      {/* Stacked cards: the same idiom the batched-event node uses for "several
          of these", so repetition reads the same way wherever it appears. */}
      <div
        className={cn(
          'bg-canvasBase absolute inset-0 translate-x-1.5 translate-y-1.5 rounded-md border',
          failing ? getStatusBorderClass('FAILED') : 'border-subtle'
        )}
      />
      <div
        className={cn(
          'bg-canvasBase absolute inset-0 translate-x-[3px] translate-y-[3px] rounded-md border',
          failing ? getStatusBorderClass('FAILED') : 'border-subtle'
        )}
      />

      <div
        className={cn(
          'bg-canvasBase absolute inset-0 flex flex-col justify-center gap-1 rounded-md border px-3 py-1.5',
          getStatusBorderClass(group.status),
          selected && SELECTED_RING,
          data.future && 'opacity-25'
        )}
      >
        <Handle type="target" position={Position.Left} className={HANDLE} />

        <div className="flex items-center gap-1.5">
          <Icon className={cn('h-3.5 w-3.5 shrink-0', getStatusTextClass(group.status))} />
          <span className="text-basis truncate text-xs font-medium" title={title}>
            {title}
          </span>
          <span
            className="text-basis bg-canvasMuted shrink-0 rounded px-1 font-mono text-[10px] tabular-nums leading-[14px]"
            title={`${group.count} repetitions of this step`}
          >
            {group.count}×
          </span>
          {/* The expander. A collapsed node has to be openable, or it is just a
              worse version of hiding the data. */}
          <button
            type="button"
            aria-label={`Expand ${group.count} steps`}
            title={`Expand ${group.count} steps`}
            onClick={(event) => {
              event.stopPropagation();
              data.onExpandGroup?.(group.id);
            }}
            className="nodrag text-muted hover:text-basis shrink-0"
          >
            <RiExpandDiagonalLine className="h-3 w-3" />
          </button>
        </div>

        <div className="flex items-center gap-1.5 overflow-hidden text-[10px] leading-none">
          {typeKey && (
            <span className="text-muted shrink-0 truncate font-mono">
              {STEP_TYPE_LABEL[typeKey]}
            </span>
          )}
          <span className="text-subtle shrink-0 font-mono">{elapsed}</span>
          {exceptions && (
            <span
              className={cn('shrink-0 truncate', failing ? 'text-status-failedText' : 'text-muted')}
              title={exceptions}
            >
              {exceptions}
            </span>
          )}
        </div>

        {variantLine ? (
          <span className="text-muted truncate text-[10px] leading-none" title={variantLine}>
            {variantLine}
          </span>
        ) : (
          <Sparkline values={group.durationsMs} className={getStatusTextClass(group.status)} />
        )}

        <Handle type="source" position={Position.Right} className={HANDLE} />
      </div>
    </div>
  );
}

export const CANVAS_NODE_TYPES = {
  canvasStep: CanvasStepNode,
  canvasJoin: CanvasJoinNode,
  canvasGroup: CanvasGroupNode,
};
