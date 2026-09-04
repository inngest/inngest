/**
 * Zoom and fit, in our own markup.
 *
 * React Flow ships a `<Controls>` whose buttons are hard-coded to a white
 * background with a white-filled icon, which is invisible against the canvas in
 * dark mode and unstyleable from here without reaching into its class names.
 * Three buttons are cheaper than fighting that, and they match the legend.
 */
import {
  RiAddLine,
  RiCollapseDiagonalLine,
  RiExpandDiagonalLine,
  RiFocus3Line,
  RiStackLine,
  RiSubtractLine,
} from '@remixicon/react';
import { useReactFlow } from '@xyflow/react';

import { cn } from '../../utils/classNames';
import type { CanvasViewMode } from './Canvas';

const FIT_OPTIONS = { padding: 0.14, maxZoom: 1.2 } as const;

const BUTTON =
  'text-subtle hover:bg-canvasMuted hover:text-basis flex h-6 w-6 items-center justify-center transition-colors';

export function CanvasControls({
  onExpand,
  mode,
  onModeChange,
  groupCount,
}: {
  onExpand?: () => void;
  /** Omitted when the run has nothing repeated, so the toggle is not offered. */
  mode?: CanvasViewMode;
  onModeChange?: (mode: CanvasViewMode) => void;
  groupCount?: number;
}) {
  const { zoomIn, zoomOut, fitView } = useReactFlow();

  return (
    <div className="border-subtle bg-canvasBase divide-subtle flex divide-x overflow-hidden rounded border">
      {mode && onModeChange && (
        <button
          type="button"
          onClick={() => onModeChange(mode === 'aggregated' ? 'expanded' : 'aggregated')}
          className={cn(BUTTON, mode === 'aggregated' && 'text-basis bg-canvasMuted')}
          aria-pressed={mode === 'aggregated'}
          title={
            mode === 'aggregated'
              ? `Showing ${groupCount} repeated step${
                  groupCount === 1 ? '' : 's'
                } collapsed. Click to unroll every step.`
              : 'Collapse repeated steps'
          }
        >
          {mode === 'aggregated' ? (
            <RiStackLine className="h-3.5 w-3.5" />
          ) : (
            <RiCollapseDiagonalLine className="h-3.5 w-3.5" />
          )}
        </button>
      )}
      <button type="button" onClick={() => void zoomIn()} className={BUTTON} title="Zoom in">
        <RiAddLine className="h-3.5 w-3.5" />
      </button>
      <button type="button" onClick={() => void zoomOut()} className={BUTTON} title="Zoom out">
        <RiSubtractLine className="h-3.5 w-3.5" />
      </button>
      <button
        type="button"
        onClick={() => void fitView(FIT_OPTIONS)}
        className={BUTTON}
        title="Fit to view"
      >
        <RiFocus3Line className="h-3.5 w-3.5" />
      </button>
      {onExpand && (
        <button type="button" onClick={onExpand} className={BUTTON} title="Expand">
          <RiExpandDiagonalLine className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  );
}
