'use client';

import { useCallback } from 'react';
import type React from 'react';

import { clamp, updateSplit } from './common';

type NotchProps = {
  containerRef: React.RefObject<HTMLDivElement>;
  minSplitPercentage: number;
  maxSplitPercentage: number;
};

export function Notch({ containerRef, minSplitPercentage, maxSplitPercentage }: NotchProps) {
  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();

      const el = containerRef.current;
      if (el === null) return;

      const onMove = makeOnMove(el, minSplitPercentage, maxSplitPercentage);

      window.addEventListener('mousemove', onMove);
      window.addEventListener(
        'mouseup',
        () => {
          window.removeEventListener('mousemove', onMove);
        },
        { once: true }
      );
    },
    [containerRef, minSplitPercentage, maxSplitPercentage]
  );

  return (
    <div
      className="absolute left-0 right-0 top-[var(--split)] z-10 flex h-6 -translate-y-1/2 cursor-row-resize items-center justify-center"
      onMouseDown={handleMouseDown}
    >
      <div className="bg-canvasMuted pointer-events-none absolute left-0 right-0 h-0.5" />
      <div className="border-muted bg-canvasSubtle pointer-events-none relative h-2 w-8 rounded-full border">
        <div className="bg-surfaceSubtle absolute left-1/2 top-1/2 h-px w-3 -translate-x-1/2 -translate-y-1/2 rounded-full" />
      </div>
    </div>
  );
}

function makeOnMove(
  el: HTMLElement,
  minSplitPercentage: number,
  maxSplitPercentage: number
): (ev: MouseEvent) => void {
  return (ev: MouseEvent) => {
    const r = el.getBoundingClientRect();
    const pct = ((ev.clientY - r.top) / r.height) * 100;
    updateSplit(el, clamp(pct, minSplitPercentage, maxSplitPercentage));
  };
}
