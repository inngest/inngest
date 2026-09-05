/**
 * The fixture gallery: every committed canvas fixture, rendered through the
 * real Canvas and Timeline.
 *
 * The fixtures already flow into vitest, which tells you whether the *model* is
 * right. This is the other half of that loop — whether the thing on screen got
 * better — over 37 shapes that are known to be hard, in under a second, without
 * running the Go server, the SDK, or a browser against a live run.
 *
 * Dev only, twice over: this app is not the Cloud dashboard, and the route
 * redirects away unless `import.meta.env.DEV`. The fixture payloads are behind
 * a dynamic import guarded on the same constant, so Rollup drops the branch and
 * the ~4MB of JSON never reaches the binary the dev server ships.
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Timeline } from '@inngest/components/RunDetailsV4/Timeline';
import { TimelineLegend } from '@inngest/components/RunDetailsV4/TimelineLegend';
import { Canvas } from '@inngest/components/RunDetailsV4/canvas/Canvas';
import type {
  CanvasFixture,
  FixtureGroup,
} from '@inngest/components/RunDetailsV4/canvas/__fixtures__/index';
import {
  applyCollapseToBars,
  planCollapse,
  shouldAggregate,
} from '@inngest/components/RunDetailsV4/canvas/collapse';
import { toCanvasGraph } from '@inngest/components/RunDetailsV4/canvas/graph';
import type { TimelineBarData } from '@inngest/components/RunDetailsV4/TimelineBar.types';
import type { Trace } from '@inngest/components/RunDetailsV4/types';
import {
  traceRollup,
  traceToTimelineData,
} from '@inngest/components/RunDetailsV4/utils/traceConversion';
import {
  traceWalk,
  useStepSelection,
} from '@inngest/components/RunDetailsV4/runDetailsUtils';
import { useSearchParam } from '@inngest/components/hooks/useSearchParams';
import { cn } from '@inngest/components/utils/classNames';
import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/_dashboard/canvas-gallery/')({
  beforeLoad: () => {
    if (!import.meta.env.DEV) {
      throw redirect({ to: '/runs' });
    }
  },
  component: CanvasGalleryComponent,
});

//
// Guarded on DEV so the whole import is statically removable in a production
// build. Do not collapse this into a bare `import(...)`.
const loadFixtures = (): Promise<CanvasFixture[]> =>
  import.meta.env.DEV
    ? import('@inngest/components/RunDetailsV4/canvas/__fixtures__/index').then(
        (m) => m.FIXTURES,
      )
    : Promise.resolve([]);

const GROUP_ORDER: FixtureGroup[] = [
  'basics',
  'waiting',
  'failure',
  'parallel',
  'adversarial',
  'lineage',
  'scale',
];

const GROUP_LABEL: Record<FixtureGroup, string> = {
  basics: 'Basics',
  waiting: 'Waiting',
  failure: 'Failure',
  parallel: 'Parallel',
  adversarial: 'Adversarial',
  lineage: 'Reported lineage',
  scale: 'Scale',
};

//
// The brief's target: no run renders more than ~40 nodes or rows at rest,
// whatever its step count. Shown on every fixture so regressions against it are
// visible rather than something you have to go and measure.
const NODE_BUDGET = 40;

function countSpans(trace: Trace): number {
  let n = 0;
  const walk = (t: Trace) => {
    n += 1;
    t.childrenSpans?.forEach(walk);
  };
  walk(trace);
  return n;
}

//
// Rows the timeline actually shows before anyone expands anything, which is
// what the budget is about. The Timeline seeds its expansion state with the
// root bars only, so this descends into a root and stops there — counting a
// collapsed group's 500 hidden members would contradict the screen.
function countBars(bars: TimelineBarData[]): number {
  return bars.reduce(
    (n, bar) => n + 1 + (bar.isRoot ? countBars(bar.children ?? []) : 0),
    0,
  );
}

function CanvasGalleryComponent() {
  const [fixtures, setFixtures] = useState<CanvasFixture[] | null>(null);
  const [fixtureID, setFixtureID] = useSearchParam('fixture');

  useEffect(() => {
    let live = true;
    void loadFixtures().then((f) => {
      if (live) setFixtures(f);
    });
    return () => {
      live = false;
    };
  }, []);

  const selected = useMemo(() => {
    if (!fixtures?.length) return null;
    return fixtures.find((f) => f.id === fixtureID) ?? fixtures[0];
  }, [fixtures, fixtureID]);

  if (!fixtures) {
    return <div className="text-muted p-6 text-sm">Loading fixtures…</div>;
  }

  return (
    <div className="bg-canvasBase flex h-full min-h-0 w-full">
      <nav className="border-muted w-64 shrink-0 overflow-y-auto border-r py-3">
        <p className="text-muted px-3 pb-2 text-[11px] uppercase tracking-wide">
          {fixtures.length} fixtures
        </p>
        {GROUP_ORDER.map((group) => {
          const inGroup = fixtures.filter((f) => f.group === group);
          if (!inGroup.length) return null;
          return (
            <div key={group} className="pb-2">
              <p className="text-muted px-3 py-1 text-[11px] font-medium">
                {GROUP_LABEL[group]}
              </p>
              {inGroup.map((f) => (
                <button
                  key={f.id}
                  type="button"
                  onClick={() => setFixtureID(f.id)}
                  aria-current={f.id === selected?.id}
                  className={cn(
                    'block w-full truncate px-3 py-1 text-left font-mono text-xs',
                    f.id === selected?.id
                      ? 'bg-secondary-3xSubtle text-basis'
                      : 'text-subtle hover:bg-canvasSubtle',
                  )}
                  title={f.note}
                >
                  {f.id}
                </button>
              ))}
            </div>
          );
        })}
      </nav>

      <div className="min-w-0 flex-1 overflow-y-auto">
        {selected ? (
          <FixtureView
            key={selected.id}
            fixture={selected}
            fixtures={fixtures}
          />
        ) : null}
      </div>
    </div>
  );
}

function FixtureView({
  fixture,
  fixtures,
}: {
  fixture: CanvasFixture;
  fixtures: CanvasFixture[];
}) {
  const runID = `fixture:${fixture.id}`;
  const raw = fixture.data.run.trace as Trace;

  const rolledUp = useMemo(() => traceRollup(raw), [raw]);

  // The Timeline reads selection from the shared emitter but does not write to
  // it — `onSelectStep` is how a host drives it, and it needs a Trace rather
  // than the bar id the row hands back. `RunDetailsV4` does exactly this; the
  // gallery did not, so clicking a row here did nothing while clicking the
  // equivalent canvas node worked, and no causal line ever persisted.
  const { selectStep } = useStepSelection({ runID });
  const traceMap = useMemo(() => {
    const map = new Map<string, Trace>();
    traceWalk(rolledUp, (t) => map.set(t.spanID, t));
    return map;
  }, [rolledUp]);
  const handleSelectStep = useCallback(
    (stepID: string) => {
      const trace = traceMap.get(stepID);
      if (trace) selectStep({ trace, runID });
    },
    [traceMap, selectStep, runID],
  );
  const graph = useMemo(() => toCanvasGraph(rolledUp), [rolledUp]);
  const timelineData = useMemo(
    () => traceToTimelineData(rolledUp, { runID, functionSlug: fixture.id }),
    [rolledUp, runID, fixture.id],
  );

  const workNodes = graph.nodes.filter(
    (n) => n.kind === 'step' || n.kind === 'wait' || n.kind === 'invoke',
  ).length;
  // What the two views actually draw, which is the number the budget is about —
  // not the uncollapsed total, which would contradict what is on screen.
  const plan = useMemo(() => planCollapse(graph), [graph]);
  const aggregated = shouldAggregate(graph, plan);
  const drawnNodes = aggregated ? plan.nodesAtRest : plan.nodesExpanded;
  // The gallery has no server, so a child run is resolved from the fixture set:
  // `invoke` and `child` were captured as a pair for exactly this. Real apps
  // supply a loader that fetches the run.
  const childTrace = useCallback(
    (childRunID: string) => {
      const candidate = fixtures.find(
        (f) =>
          (f.data.run.trace as Trace).spanID === childRunID || f.id === 'child',
      );
      return candidate ? traceRollup(candidate.data.run.trace as Trace) : null;
    },
    [fixtures],
  );

  const loadChildRun = useCallback(
    async (childRunID: string) => {
      const rolled = childTrace(childRunID);
      if (!rolled) return null;
      return traceToTimelineData(rolled, { runID: childRunID }).bars;
    },
    [childTrace],
  );

  // The canvas wants the child's trace rather than its rows — it builds the
  // child's own graph from it and draws that inside the invoke node.
  const loadChildTrace = useCallback(
    async (childRunID: string) => childTrace(childRunID),
    [childTrace],
  );

  const rows = countBars(
    aggregated
      ? applyCollapseToBars(timelineData.bars, plan)
      : timelineData.bars,
  );

  return (
    <div className="flex flex-col gap-3 p-4">
      <header className="flex flex-col gap-1">
        <div className="flex items-baseline gap-2">
          <h1 className="text-basis font-mono text-sm">{fixture.id}</h1>
          <span className="text-muted text-xs">{fixture.title}</span>
          <span className="text-muted border-subtle rounded border px-1 text-[10px]">
            SDK {fixture.sdk}
          </span>
        </div>
        <p className="text-subtle max-w-3xl text-xs leading-relaxed">
          {fixture.note}
        </p>
        {fixture.synthetic && (
          <p className="text-warning max-w-3xl text-xs leading-relaxed">
            Synthetic — not captured from a real run. {fixture.synthetic}
          </p>
        )}
      </header>

      <dl className="text-subtle flex flex-wrap gap-x-4 gap-y-1 font-mono text-[11px] tabular-nums">
        <Fact label="status" value={fixture.data.run.status} />
        <Fact label="spans" value={countSpans(raw)} />
        <Fact label="levels (full)" value={graph.levels.length} />
        <Fact label="work nodes (full)" value={workNodes} />
        <Fact
          label="nodes at rest"
          value={`${drawnNodes} / ${NODE_BUDGET}`}
          tone={drawnNodes > NODE_BUDGET ? 'over' : 'ok'}
        />
        {plan.groups.length > 0 && (
          <Fact
            label="collapsed"
            value={`${plan.groups.length} group${
              plan.groups.length === 1 ? '' : 's'
            }${aggregated ? '' : ' (off)'}`}
          />
        )}
        <Fact label="edges (full)" value={graph.edges.length} />
        <Fact
          label="rows at rest"
          value={`${rows} / ${NODE_BUDGET}`}
          tone={rows > NODE_BUDGET ? 'over' : 'ok'}
        />
        {/* The same word the canvas itself uses. It said "exact" in its corner
            while this strip said "sdk" — two labels for one thing on one
            screen, and the reader has to work out they are the same. */}
        <Fact
          label="grouping"
          value={
            graph.parallelismSource === 'sdk'
              ? 'exact'
              : graph.parallelismSource
          }
        />
        <Fact label="branches resolved" value={graph.branchesResolved} />
      </dl>

      {graph.warnings.length > 0 && (
        <ul className="text-subtle text-[11px] leading-relaxed">
          {graph.warnings.map((w) => (
            <li key={w}>· {w}</li>
          ))}
        </ul>
      )}

      <Canvas trace={rolledUp} runID={runID} loadChildRun={loadChildTrace} />

      <div className="border-muted border-t pt-2">
        <div className="flex justify-end pb-1">
          <TimelineLegend />
        </div>
        <Timeline
          loadChildRun={loadChildRun}
          data={timelineData}
          runID={runID}
          collapse={aggregated ? plan : undefined}
          onSelectStep={handleSelectStep}
        />
      </div>
    </div>
  );
}

function Fact({
  label,
  value,
  tone,
}: {
  label: string;
  value: string | number;
  tone?: 'ok' | 'over';
}) {
  return (
    <div className="flex gap-1">
      <dt className="text-muted">{label}</dt>
      <dd className={cn(tone === 'over' ? 'text-error' : 'text-basis')}>
        {value}
      </dd>
    </div>
  );
}
