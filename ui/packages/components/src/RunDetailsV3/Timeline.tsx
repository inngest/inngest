import { useCallback, useEffect, useRef, useState } from 'react';
import { max, min } from 'date-fns';

import { toMaybeDate } from '../utils/date';
import { isLazyDone, type Lazy } from '../utils/lazyLoad';
import { TimingLegend } from './TimingLegend';
import { Trace } from './Trace';
import { traceWalk } from './utils';

type Props = {
  runID: string;
  trace: Lazy<React.ComponentProps<typeof Trace>['trace']>;
};

export const Timeline = ({ runID, trace }: Props) => {
  const [leftWidth, setLeftWidth] = useState(30);
  const [isDragging, setIsDragging] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const onResizeStart = useCallback(() => {
    setIsDragging(true);
  }, []);

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isDragging) return;

      const container = containerRef.current;
      if (!container) return;

      const containerRect = container.getBoundingClientRect();
      const newWidth = ((e.clientX - containerRect.left) / containerRect.width) * 100;
      setLeftWidth(Math.min(Math.max(newWidth, 20), 80));
    },
    [isDragging]
  );

  useEffect(() => {
    if (!isDragging) return;

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging, handleMouseMove, handleMouseUp]);

  if (!isLazyDone(trace)) {
    // TODO: Properly handle loading state
    return null;
  }

  let minTime = new Date(trace.queuedAt);
  let maxTime = toMaybeDate(trace.endedAt) ?? new Date();

  traceWalk(trace, (t) => {
    minTime = min([minTime, new Date(t.queuedAt)]);

    const endedAt = toMaybeDate(t.endedAt);
    if (endedAt) {
      maxTime = max([endedAt, maxTime]);
    }
  });

  return (
    <div className="w-full pb-4 pr-8" ref={containerRef}>
      {/* Timing Legend (US3 - EXE-1217) */}
      <div className="mb-3 flex justify-end pr-2">
        <TimingLegend />
      </div>

      <Trace
        depth={0}
        maxTime={maxTime}
        minTime={minTime}
        runID={runID}
        trace={{ ...trace, name: 'Run' }}
        leftWidth={leftWidth}
        onResizeStart={onResizeStart}
      />
    </div>
  );
};
