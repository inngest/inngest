/**
 * A small uppercase tag naming what a row is, like JOB, MACHINE or AGENT.
 */

import { cn } from '../utils/classNames';

/** The kind is free-form text from the caller, shown in upper case */
export function SpanKindTag({ kind, className }: { kind: string; className?: string }) {
  return (
    <span
      data-testid="span-kind-tag"
      className={cn(
        'bg-info text-info inline-block shrink-0 rounded px-1.5 align-middle font-sans text-[0.66rem] font-semibold leading-4 tracking-[0.04em]',
        className
      )}
    >
      {kind.toUpperCase()}
    </span>
  );
}
