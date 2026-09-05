/**
 * The Run Canvas: a left-to-right flow graph of a single run.
 *
 * Selection is delegated to the existing step-selection emitter, so clicking a
 * node opens the same right-hand panel the timeline opens — and selecting a row
 * in the timeline highlights the matching node here. This component owns no
 * step state of its own.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { RiShieldCheckLine } from '@remixicon/react';
import {
  Background,
  BackgroundVariant,
  Panel,
  ReactFlow,
  ReactFlowProvider,
  useNodesInitialized,
  useReactFlow,
  type Node,
} from '@xyflow/react';

import '@xyflow/react/dist/style.css';
import { useQuery } from '@tanstack/react-query';

import { Modal } from '../../Modal/Modal';
import { cn } from '../../utils/classNames';
import { traceWalk, useStepHover, useStepSelection } from '../runDetailsUtils';
import type { Trace } from '../types';
import { runValue, valueLines } from '../value';
import { CanvasControls } from './CanvasControls';
import { CanvasLegend } from './CanvasLegend';
import { CANVAS_NODE_TYPES } from './CanvasNode';
import { applyCollapse, planCollapse, shouldAggregate } from './collapse';
import { toCanvasGraph } from './graph';
import type { CanvasGraph, CanvasTrigger } from './graph.types';
import { toFlowElements, type CanvasNodeData } from './toFlowElements';

export type CanvasViewMode = 'aggregated' | 'expanded';

type Props = {
  /** The rolled-up trace, as the timeline uses. */
  trace: Trace;
  runID: string;
  /** Same loader the run header uses; supplies the triggering event(s). */
  getTrigger?: (runID: string) => Promise<CanvasTrigger>;
  /**
   * Set on the copy inside the expanded modal, which fills the viewport and has
   * nothing left to expand into.
   */
  expanded?: boolean;
  /**
   * Fetches the run a `step.invoke` started, so it can be shown inside the
   * invoke node rather than only behind a link.
   *
   * Called only when a node is actually opened, and once per child. Omit it and
   * invoke nodes carry no disclosure at all — an affordance that does nothing is
   * worse than none.
   */
  loadChildRun?: (childRunID: string) => Promise<Trace | null>;
  /** Leaves this run for the child's own page. */
  onOpenChildRun?: (childRunID: string) => void;
};

/**
 * Deliberately no `minZoom`.
 *
 * A wide graph in a short pane fits only by zooming out, and on `chains` —
 * seven levels in 950px — that lands at ~0.55, rendering `step.run 32ms` at
 * about six pixels. A floor was tried and is worse: at 0.75 the Trigger and the
 * run's outcome both fall off the ends, and a reader who cannot see how the run
 * started or how it finished has lost more than one who has to squint.
 *
 * Nothing here recovers the empty top and bottom of the pane — the zoom is
 * bound by width, so vertical space cannot help. The real answer for a
 * wide-and-shallow run is more horizontal space, which is what the full-screen
 * control is for.
 */
const FIT_OPTIONS = { padding: 0.14, maxZoom: 1.2 } as const;

/**
 * Refits whenever the graph or the pane size changes. The canvas can mount into
 * a collapsed container, so the initial `fitView` has nothing useful to measure
 * against and the graph renders clipped without this.
 */
function FitView({
  graphKey,
  paneRef,
}: {
  graphKey: string;
  paneRef: React.RefObject<HTMLElement>;
}) {
  const { fitView } = useReactFlow();
  const initialised = useNodesInitialized();

  useEffect(() => {
    if (initialised) void fitView(FIT_OPTIONS);
  }, [initialised, graphKey, fitView]);

  useEffect(() => {
    const pane = paneRef.current;
    if (!pane) return;
    const observer = new ResizeObserver(() => void fitView(FIT_OPTIONS));
    observer.observe(pane);
    return () => observer.disconnect();
  }, [paneRef, fitView]);

  return null;
}

export function Canvas(props: Props) {
  return (
    <ReactFlowProvider>
      <CanvasInner {...props} />
    </ReactFlowProvider>
  );
}

/**
 * The same canvas, filling the window.
 *
 * A wide fan-out or a long agent loop fits into 300px only by zooming out until
 * the nodes are unreadable, so the graph needs somewhere bigger to be read and
 * clicked through. Selection is shared, so a step picked in here is still
 * selected behind it.
 */
function ExpandedCanvas({ onClose, ...props }: Props & { onClose: () => void }) {
  return (
    <Modal isOpen onClose={onClose} className="h-[85vh] w-[92vw] max-w-none p-0">
      <div className="h-full w-full p-3">
        <Canvas {...props} expanded />
      </div>
    </Modal>
  );
}

function CanvasInner({ trace, runID, getTrigger, expanded, loadChildRun, onOpenChildRun }: Props) {
  const paneRef = useRef<HTMLDivElement>(null);
  const [showExpanded, setShowExpanded] = useState(false);

  const { data: trigger } = useQuery({
    queryKey: ['run-trigger', runID],
    queryFn: () => getTrigger!(runID),
    enabled: Boolean(getTrigger),
    retry: 3,
  });

  const graph = useMemo(() => toCanvasGraph(trace, trigger), [trace, trigger]);

  const { selectedStep, selectStep } = useStepSelection({ runID });
  const selectedSpanID = selectedStep?.trace.spanID;

  // One subscriber for the whole canvas, matching the timeline. Hovering a node
  // highlights the matching row below and vice versa.
  const { hoveredSpanID, hoverStep } = useStepHover({ runID });

  const plan = useMemo(() => planCollapse(graph), [graph]);

  // What the platform did for this run that the user would otherwise have built
  // or lost. Factual and quantified, or absent — a plain run gets a plain page.
  const value = useMemo(
    () => valueLines(runValue(trace, trigger?.IDs?.length ?? 1)),
    [trace, trigger]
  );

  // Two modes rather than one cleverer layout, which is what everyone who has
  // confronted this shipped. Aggregated only becomes the default when the run is
  // actually too big to read; a small run has nothing to gain from it and would
  // just be hiding itself for no reason.
  const [mode, setMode] = useState<CanvasViewMode>(() =>
    shouldAggregate(graph, plan) ? 'aggregated' : 'expanded'
  );
  const [openGroups, setOpenGroups] = useState<ReadonlySet<string>>(new Set());

  const expandGroup = useCallback((groupID: string) => {
    setOpenGroups((open) => new Set(open).add(groupID));
  }, []);

  // Expanding a group must be reversible. Without this, opening one on a
  // 40-iteration loop drops the fit to a grey hairline — the exact state
  // collapsing exists to prevent — with no way back but a page reload.
  //
  // So the toolbar control does the nearest useful thing rather than blindly
  // flipping: with groups open it re-collapses them and stays aggregated, and
  // only switches mode once there is nothing left to close.
  const changeMode = useCallback(
    (next: CanvasViewMode) => {
      if (mode === 'aggregated' && openGroups.size > 0) {
        setOpenGroups(new Set());
        return;
      }
      setOpenGroups(new Set());
      setMode(next);
    },
    [mode, openGroups]
  );

  const { graph: shown, groupByNodeID } = useMemo(
    () =>
      mode === 'expanded'
        ? { graph, groupByNodeID: new Map() }
        : applyCollapse(graph, plan, openGroups),
    [graph, plan, mode, openGroups]
  );

  const collapse = useMemo(
    () => ({ groupByNodeID, onExpandGroup: expandGroup }),
    [groupByNodeID, expandGroup]
  );

  // A `step.invoke` started another run, and that run is the answer to what the
  // invoke did. It is fetched only when someone opens it — a tree of invokes
  // must not pull the world down on first paint — and kept for the life of the
  // view once it arrives.
  const [openChildren, setOpenChildren] = useState<ReadonlySet<string>>(new Set());
  const [childGraphs, setChildGraphs] = useState<ReadonlyMap<string, CanvasGraph | null>>(
    new Map()
  );
  const [loadingChildren, setLoadingChildren] = useState<ReadonlySet<string>>(new Set());

  const toggleChild = useCallback(
    (nodeID: string) => {
      const node = shown.nodes.find((n) => n.id === nodeID);
      const childRunID = node?.childRunID;
      if (!childRunID) return;

      setOpenChildren((open) => {
        const next = new Set(open);
        if (next.has(nodeID)) {
          next.delete(nodeID);
          return next;
        }
        next.add(nodeID);
        return next;
      });

      // Fetch once. A second open of the same child reuses what came back,
      // including a failure — retrying silently on every toggle would hammer a
      // run that is genuinely gone.
      if (!loadChildRun || childGraphs.has(childRunID)) return;

      setLoadingChildren((l) => new Set(l).add(childRunID));
      void loadChildRun(childRunID)
        .then((childTrace) => {
          setChildGraphs((g) =>
            new Map(g).set(childRunID, childTrace ? toCanvasGraph(childTrace) : null)
          );
        })
        .catch(() => {
          setChildGraphs((g) => new Map(g).set(childRunID, null));
        })
        .finally(() => {
          setLoadingChildren((l) => {
            const next = new Set(l);
            next.delete(childRunID);
            return next;
          });
        });
    },
    [shown, loadChildRun, childGraphs]
  );

  const children = useMemo(
    () =>
      // Depth-capped at one. The preview inside a node shows the child's shape,
      // not its children's — a tree of invokes drawn inside itself is neither
      // readable nor bounded. "Open" takes the reader to the child's own page,
      // where its invokes expand in turn.
      loadChildRun
        ? {
            open: openChildren,
            graphs: childGraphs,
            loading: loadingChildren,
            onToggle: toggleChild,
            onOpenRun: onOpenChildRun,
          }
        : undefined,
    [loadChildRun, openChildren, childGraphs, loadingChildren, toggleChild, onOpenChildRun]
  );

  const { nodes, edges } = useMemo(
    () => toFlowElements(shown, selectedSpanID, collapse, hoveredSpanID, children),
    [shown, selectedSpanID, collapse, hoveredSpanID, children]
  );

  const traceMap = useMemo(() => {
    const map = new Map<string, Trace>();
    traceWalk(trace, (t) => map.set(t.spanID, t));
    return map;
  }, [trace]);

  const onNodeMouseEnter = useCallback(
    (_: unknown, node: Node<CanvasNodeData>) => {
      hoverStep({ spanID: node.data.spanID, runID });
    },
    [hoverStep, runID]
  );

  const onNodeMouseLeave = useCallback(() => hoverStep(undefined), [hoverStep]);

  const onNodeClick = useCallback(
    (_: unknown, node: Node<CanvasNodeData>) => {
      const selected = traceMap.get(node.data.spanID);
      if (selected) selectStep({ trace: selected, runID });
    },
    [traceMap, selectStep, runID]
  );

  return (
    <div className={cn('flex flex-col gap-1.5', expanded && 'h-full')}>
      <div
        ref={paneRef}
        className={cn(
          'border-subtle bg-canvasSubtle w-full rounded border',
          expanded ? 'min-h-0 flex-1' : 'h-[300px]'
        )}
      >
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={CANVAS_NODE_TYPES}
          onNodeClick={onNodeClick}
          onNodeMouseEnter={onNodeMouseEnter}
          onNodeMouseLeave={onNodeMouseLeave}
          fitView
          fitViewOptions={FIT_OPTIONS}
          // React Flow's default floor is 0.5, which is not far enough out for a
          // wide fan-out or a long agent loop to fit at all.
          minZoom={0.05}
          proOptions={{ hideAttribution: true }}
          nodesDraggable={false}
          nodesConnectable={false}
          className="bg-canvasSubtle"
        >
          <Background variant={BackgroundVariant.Dots} gap={16} size={1} className="opacity-40" />

          <Panel position="top-left" className="!m-2">
            <CanvasLegend />
          </Panel>

          <Panel position="top-right" className="!m-2">
            <CanvasControls
              onExpand={expanded ? undefined : () => setShowExpanded(true)}
              mode={plan.groups.length ? mode : undefined}
              onModeChange={changeMode}
              openGroupCount={openGroups.size}
              groupCount={plan.groups.length}
            />
          </Panel>
          <FitView graphKey={`${runID}:${shown.nodes.length}`} paneRef={paneRef} />
        </ReactFlow>
      </div>

      {showExpanded && (
        <ExpandedCanvas
          trace={trace}
          runID={runID}
          getTrigger={getTrigger}
          onClose={() => setShowExpanded(false)}
        />
      )}

      {value.length > 0 && (
        <ul className="text-subtle flex flex-wrap gap-x-3 px-1 text-[11px] leading-relaxed">
          {value.map((line) => (
            <li key={line} className="flex items-center gap-1">
              <RiShieldCheckLine className="text-muted h-3 w-3 shrink-0" />
              {line}
            </li>
          ))}
        </ul>
      )}

      <div className="flex items-start justify-between gap-3 px-1">
        {/* Bulleted only when there is more than one thing to say. A single
            line prefixed with a lone `·` reads as a stray character rather than
            as a list, which is how `wide`'s one warning looked. */}
        <ul className="text-subtle text-[11px] leading-relaxed">
          {shown.warnings.map((w) => (
            <li key={w}>{shown.warnings.length > 1 ? `· ${w}` : w}</li>
          ))}
        </ul>
        {graph.parallelismSource !== 'none' && (
          <span
            className="text-muted shrink-0 text-[11px] leading-relaxed"
            title={
              graph.parallelismSource === 'sdk'
                ? 'The SDK reported which steps it planned together, so the grouping is exact.'
                : 'This SDK reports one step per response, so grouping is inferred from execution overlap.'
            }
          >
            {graph.parallelismSource === 'sdk' ? 'grouping: exact' : 'grouping: inferred'}
          </span>
        )}
      </div>
    </div>
  );
}
