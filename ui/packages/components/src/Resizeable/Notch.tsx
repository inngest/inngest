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
      className="absolute left-0 right-0 top-[var(--split)] -translate-y-1/2 cursor-row-resize"
      onMouseDown={handleMouseDown}
    >
      <div className="bg-border mx-auto h-1 w-12 rounded-full" />
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
