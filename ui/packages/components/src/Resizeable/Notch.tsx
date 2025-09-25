'use client';

import { useCallback } from 'react';
import type React from 'react';

import { cn } from '../utils/classNames';
import { clamp, updateSplit } from './common';
import type { Orientation } from './types';

type NotchProps = {
  containerRef: React.RefObject<HTMLDivElement>;
  maxSplitPercentage: number;
  minSplitPercentage: number;
  orientation: Orientation;
};

export function Notch({
  containerRef,
  maxSplitPercentage,
  minSplitPercentage,
  orientation,
}: NotchProps) {
  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();

      const el = containerRef.current;
      if (el === null) return;

      const onMove = makeOnMove(el, { minSplitPercentage, maxSplitPercentage, orientation });

      window.addEventListener('mousemove', onMove);
      window.addEventListener(
        'mouseup',
        () => {
          window.removeEventListener('mousemove', onMove);
        },
        { once: true }
      );
    },
    [containerRef, minSplitPercentage, maxSplitPercentage, orientation]
  );

  return (
    <div
      className={cn(
        'absolute z-10 flex items-center justify-center',
        buildCrossAxisSizeClass(orientation),
        buildTrackClasses(orientation)
      )}
      onMouseDown={handleMouseDown}
    >
      <div
        className={cn(
          'bg-canvasMuted pointer-events-none absolute',
          buildDividerClasses(orientation)
        )}
      />
      <div
        className={cn(
          'bg-canvasSubtle border-muted pointer-events-none relative rounded-full border',
          buildHandleSizeClasses(orientation)
        )}
      >
        <div
          className={cn(
            'bg-surfaceSubtle absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full',
            buildHandleStripeClasses(orientation)
          )}
        />
      </div>
    </div>
  );
}

type OnMoveOptions = {
  maxSplitPercentage: number;
  minSplitPercentage: number;
  orientation: Orientation;
};

function makeOnMove(el: HTMLElement, options: OnMoveOptions): (ev: MouseEvent) => void {
  const { maxSplitPercentage, minSplitPercentage, orientation } = options;

  return (ev: MouseEvent) => {
    const pct = computeSplitPercentageFromEvent(el, ev, orientation);
    updateSplit(el, clamp(pct, minSplitPercentage, maxSplitPercentage));
  };
}

function buildCrossAxisSizeClass(orientation: Orientation): string {
  return orientation === 'vertical' ? 'h-6' : 'w-6';
}

function buildDividerClasses(orientation: Orientation): string {
  return orientation === 'vertical' ? 'left-0 right-0 h-0.5' : 'top-0 bottom-0 w-0.5';
}

function buildHandleSizeClasses(orientation: Orientation): string {
  return orientation === 'vertical' ? 'h-2 w-8' : 'h-8 w-2';
}

function buildHandleStripeClasses(orientation: Orientation): string {
  return orientation === 'vertical' ? 'h-px w-3' : 'h-3 w-px';
}

function buildTrackClasses(orientation: Orientation): string {
  return orientation === 'vertical'
    ? 'left-0 right-0 top-[var(--split)] -translate-y-1/2 cursor-row-resize'
    : 'top-0 bottom-0 left-[var(--split)] -translate-x-1/2 cursor-col-resize';
}

function computeSplitPercentageFromEvent(
  el: HTMLElement,
  ev: MouseEvent,
  orientation: Orientation
): number {
  const r = el.getBoundingClientRect();
  return orientation === 'vertical'
    ? ((ev.clientY - r.top) / r.height) * 100
    : ((ev.clientX - r.left) / r.width) * 100;
}
