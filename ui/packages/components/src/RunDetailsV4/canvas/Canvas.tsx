/**
 * The Run Canvas: a left-to-right flow graph of a single run.
 *
 * Selection is delegated to the existing step-selection emitter, so clicking a
 * node opens the same right-hand panel the timeline opens — and selecting a row
 * in the timeline highlights the matching node here. This component owns no
 * step state of its own.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
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
import { traceWalk, useStepSelection } from '../runDetailsUtils';
import type { Trace } from '../types';
import { CanvasControls } from './CanvasControls';
import { CanvasLegend } from './CanvasLegend';
import { CANVAS_NODE_TYPES } from './CanvasNode';
import { toCanvasGraph } from './graph';
import type { CanvasTrigger } from './graph.types';
import { toFlowElements, type CanvasNodeData } from './toFlowElements';

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
};

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

function CanvasInner({ trace, runID, getTrigger, expanded }: Props) {
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

  const { nodes, edges } = useMemo(
    () => toFlowElements(graph, selectedSpanID),
    [graph, selectedSpanID]
  );

  const traceMap = useMemo(() => {
    const map = new Map<string, Trace>();
    traceWalk(trace, (t) => map.set(t.spanID, t));
    return map;
  }, [trace]);

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
            <CanvasControls onExpand={expanded ? undefined : () => setShowExpanded(true)} />
          </Panel>
          <FitView graphKey={`${runID}:${graph.nodes.length}`} paneRef={paneRef} />
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

      <div className="flex items-start justify-between gap-3 px-1">
        <ul className="text-subtle text-[11px] leading-relaxed">
          {graph.warnings.map((w) => (
            <li key={w}>· {w}</li>
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
