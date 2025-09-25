'use client';

import { useLayoutEffect, useRef, type ReactNode } from 'react';

import { cn } from '../utils/classNames';
import { Notch } from './Notch';
import { clamp, updateSplit } from './common';
import type { Orientation } from './types';

export type ResizeableProps = {
  defaultSplitPercentage?: number;
  first: ReactNode;
  maxSplitPercentage?: number;
  minSplitPercentage?: number;
  orientation: Orientation;
  second: ReactNode;
};

export function Resizeable({
  defaultSplitPercentage = 50,
  first,
  maxSplitPercentage = 100,
  minSplitPercentage = 0,
  orientation,
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
      <div className={cn('flex h-full w-full', buildDirectionClass(orientation))}>
        <div className={cn(buildPaneMinClass(orientation), 'shrink-0 grow-0 basis-[var(--split)]')}>
          {first}
        </div>
        <div className={cn(buildPaneMinClass(orientation), 'flex-1')}>{second}</div>
      </div>
      <Notch
        containerRef={containerRef}
        maxSplitPercentage={maxSplitPercentage}
        minSplitPercentage={minSplitPercentage}
        orientation={orientation}
      />
    </div>
  );
}

function buildDirectionClass(orientation: Orientation): string {
  return orientation === 'vertical' ? 'flex-col' : 'flex-row';
}

function buildPaneMinClass(orientation: Orientation): string {
  return orientation === 'vertical' ? 'min-h-0' : 'min-w-0';
}
