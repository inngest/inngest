import { useCallback } from 'react';
import type React from 'react';

import { cn } from '../utils/classNames';
import { addSplitListeners, makeOnMove, makeOnStopDrag } from './split';
import type { Orientation } from './types';

type NotchProps = {
  containerRef: React.RefObject<HTMLDivElement>;
  maxSplitPercentage: number;
  minSplitPercentage: number;
  orientation: Orientation;
  splitKey?: string;
};

export function Notch({
  containerRef,
  maxSplitPercentage,
  minSplitPercentage,
  orientation,
  splitKey,
}: NotchProps) {
  const handlePointerDown = useCallback(
    (e: React.PointerEvent) => {
      e.preventDefault();

      const el = containerRef.current;
      if (el === null) return;

      const onMove = makeOnMove(el, { minSplitPercentage, maxSplitPercentage, orientation });
      const onStop = makeOnStopDrag(el, {
        maxSplitPercentage,
        minSplitPercentage,
        onMove,
        orientation,
        splitKey,
      });

      addSplitListeners(onMove, onStop);
    },
    [containerRef, maxSplitPercentage, minSplitPercentage, orientation, splitKey]
  );

  return (
    <div
      className={cn(
        'pointer-events-none absolute z-10 flex items-center justify-center',
        buildCrossAxisSizeClass(orientation),
        buildTrackClasses(orientation)
      )}
    >
      <div
        className={cn(
          'bg-canvasMuted absolute',
          buildDividerClasses(orientation)
        )}
      />
      <div
        className={cn(
          'bg-canvasSubtle border-muted pointer-events-auto relative rounded-full border',
          buildHandleSizeClasses(orientation),
          buildHandleCursorClass(orientation)
        )}
        onPointerDown={handlePointerDown}
      >
        <div
          className={cn(
            'bg-surfaceSubtle border-muted pointer-events-none absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border',
            buildHandleStripeClasses(orientation)
          )}
        />
      </div>
    </div>
  );
}

function buildCrossAxisSizeClass(orientation: Orientation): string {
  return orientation === 'vertical' ? 'h-2' : 'w-2';
}

function buildDividerClasses(orientation: Orientation): string {
  return orientation === 'vertical' ? 'left-0 right-0 h-px' : 'top-0 bottom-0 w-px';
}

function buildHandleSizeClasses(orientation: Orientation): string {
  return orientation === 'vertical' ? 'h-2 w-8' : 'h-8 w-2';
}

function buildHandleStripeClasses(orientation: Orientation): string {
  return orientation === 'vertical' ? 'h-px w-3' : 'h-3 w-px';
}

function buildHandleCursorClass(orientation: Orientation): string {
  return orientation === 'vertical' ? 'cursor-row-resize' : 'cursor-col-resize';
}

function buildTrackClasses(orientation: Orientation): string {
  return orientation === 'vertical'
    ? 'left-0 right-0 top-[var(--inngest-resizable-split,50%)] -translate-y-1/2'
    : 'top-0 bottom-0 left-[var(--inngest-resizable-split,50%)] -translate-x-1/2';
}
