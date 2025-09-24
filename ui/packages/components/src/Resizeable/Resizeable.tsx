'use client';

import { useLayoutEffect, useRef, type ReactNode } from 'react';

import { Notch } from './Notch';
import { clamp, updateSplit } from './common';

export type ResizeableProps = {
  defaultSplitPercentage?: number;
  first: ReactNode;
  second: ReactNode;
  minSplitPercentage?: number;
  maxSplitPercentage?: number;
};

export function Resizeable({
  defaultSplitPercentage = 37.5,
  first,
  maxSplitPercentage = 100,
  minSplitPercentage = 0,
  second,
}: ResizeableProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const el = containerRef.current;
    if (el === null) return;

    updateSplit(el, clamp(defaultSplitPercentage, minSplitPercentage, maxSplitPercentage));
  }, [defaultSplitPercentage, minSplitPercentage, maxSplitPercentage]);

  return (
    <div ref={containerRef} className="relative h-full w-full overflow-hidden">
      <div className="flex h-full w-full flex-col">
        <div className="min-h-0 shrink-0 grow-0 basis-[var(--split)]">{first}</div>
        <div className="min-h-0 flex-1">{second}</div>
      </div>
      <Notch
        containerRef={containerRef}
        minSplitPercentage={minSplitPercentage}
        maxSplitPercentage={maxSplitPercentage}
      />
    </div>
  );
}
