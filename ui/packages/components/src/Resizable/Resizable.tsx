import { useLayoutEffect, useRef, type ReactNode } from 'react';

import { cn } from '../utils/classNames';
import { Notch } from './Notch';
import { initializeSplitFromStorage } from './split';
import type { Orientation } from './types';

export type ResizableProps = {
  defaultSplitPercentage?: number;
  first: ReactNode;
  maxSplitPercentage?: number;
  minSplitPercentage?: number;
  orientation: Orientation;
  // null/undefined collapses the second pane (and the drag notch), giving
  // `first` the full width/height -- lets a caller keep `first` mounted at a
  // stable position in the tree while toggling a side panel on and off,
  // instead of conditionally swapping between <Resizable> and `first` alone
  // (which would remount `first` and any state it owns).
  second?: ReactNode | null;
  splitKey?: string;
};

export function Resizable({
  defaultSplitPercentage = 50,
  first,
  maxSplitPercentage = 100,
  minSplitPercentage = 0,
  orientation,
  second,
  splitKey,
}: ResizableProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const hasSecondPane = second != null;

  useLayoutEffect(() => {
    const el = containerRef.current;
    if (el === null || !hasSecondPane) return;

    initializeSplitFromStorage(el, {
      defaultSplitPercentage,
      maxSplitPercentage,
      minSplitPercentage,
      splitKey,
    });
  }, [defaultSplitPercentage, minSplitPercentage, maxSplitPercentage, splitKey, hasSecondPane]);

  return (
    <div ref={containerRef} className="relative h-full w-full overflow-hidden">
      <div className={cn('flex h-full w-full', buildDirectionClass(orientation))}>
        <div
          className={cn(
            buildPaneMinClass(orientation),
            hasSecondPane ? 'shrink-0 grow-0 basis-[var(--inngest-resizable-split,50%)]' : 'flex-1'
          )}
        >
          {first}
        </div>
        {hasSecondPane && (
          <div className={cn(buildPaneMinClass(orientation), 'flex-1')}>{second}</div>
        )}
      </div>
      {hasSecondPane && (
        <Notch
          containerRef={containerRef}
          maxSplitPercentage={maxSplitPercentage}
          minSplitPercentage={minSplitPercentage}
          orientation={orientation}
          splitKey={splitKey}
        />
      )}
    </div>
  );
}

function buildDirectionClass(orientation: Orientation): string {
  return orientation === 'vertical' ? 'flex-col' : 'flex-row';
}

function buildPaneMinClass(orientation: Orientation): string {
  return orientation === 'vertical' ? 'min-h-0' : 'min-w-0';
}
