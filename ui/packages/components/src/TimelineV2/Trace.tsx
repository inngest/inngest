import { useEffect, useState } from 'react';

import { CodeBlock } from '../CodeBlock';
import { cn } from '../utils/classNames';
import { InlineSpans } from './InlineSpans';
import { TraceHeading } from './TraceHeading';
import { TraceInfo } from './TraceInfo';
import type { Trace } from './types';
import { createSpanWidths } from './utils';

type Props = {
  depth: number;
  getOutput: (outputID: string) => Promise<string | null>;
  isExpandable?: boolean;
  minTime?: Date;
  maxTime?: Date;
  trace: Trace;
};

export function Trace({ depth, getOutput, isExpandable = true, maxTime, minTime, trace }: Props) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [output, setOutput] = useState<string>();

  useEffect(() => {
    if (isExpanded && !output && trace.outputID) {
      getOutput(trace.outputID).then((data) => {
        setOutput(data ?? undefined);
      });
    }
  }, [isExpanded, output]);

  if (!minTime) {
    minTime = new Date(trace.queuedAt);
  }

  if (!maxTime) {
    maxTime = new Date(trace.endedAt ?? new Date());
  }

  const widths = createSpanWidths({
    minTime,
    maxTime,
    trace: {
      endedAt: trace.endedAt ? new Date(trace.endedAt) : new Date(),
      queuedAt: new Date(trace.queuedAt),
      startedAt: trace.startedAt ? new Date(trace.startedAt) : null,
    },
  });

  let spans = [trace];
  if (!trace.isRoot && trace.childrenSpans && trace.childrenSpans.length > 0) {
    spans = trace.childrenSpans;
  }

  return (
    <div
      className={cn(
        'py-2',
        // We don't want borders or horizontal padding on step attempts
        depth === 0 && 'border-b border-slate-300 px-4',
        isExpanded && 'bg-blue-50'
      )}
    >
      <div className="flex gap-2">
        <div
          className={cn(
            // Steps and attempts need different widths, since attempts are
            // indented
            depth === 0 && 'w-72',
            depth === 1 && 'w-64'
          )}
        >
          <TraceHeading
            isExpandable={isExpandable}
            isExpanded={isExpanded}
            onClickExpandToggle={() => setIsExpanded((prev) => !prev)}
            trace={trace}
          />
        </div>

        <InlineSpans
          className="my-2"
          maxTime={maxTime}
          minTime={minTime}
          spans={spans}
          widths={widths}
        />
      </div>

      {isExpanded && (
        <div className="ml-8">
          <TraceInfo className="my-4 grow" trace={trace} />

          {output && (
            <div className="mb-4">
              <CodeBlock
                tabs={[
                  {
                    label: 'Output',
                    content: output,
                  },
                ]}
              />
            </div>
          )}

          {trace.childrenSpans?.map((child, i) => {
            return (
              <div className="flex">
                <div className="grow">
                  <Trace
                    depth={depth + 1}
                    getOutput={getOutput}
                    maxTime={maxTime}
                    minTime={minTime}
                    trace={child}
                  />
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
