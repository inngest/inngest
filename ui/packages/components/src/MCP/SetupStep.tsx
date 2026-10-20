import type { ReactNode } from 'react';

import { cn } from '../utils/classNames';

// Shared by the MCP setup page and its compact Runs popover instructions.
export function SetupStep({
  children,
  isLast = false,
  number,
  title,
  compact = false,
}: {
  children?: ReactNode;
  isLast?: boolean;
  number: number;
  title: string;
  compact?: boolean;
}) {
  return (
    <div className={cn('flex', compact ? 'gap-2' : 'gap-4')}>
      <div className="flex flex-col items-center">
        <div
          className={cn(
            'border-subtle bg-canvasBase text-basis flex shrink-0 items-center justify-center rounded-full border text-xs font-medium',
            compact ? 'h-5 w-5' : 'h-7 w-7'
          )}
        >
          {number}
        </div>
        {!isLast && <div className="border-subtle my-1 w-px flex-1 border-l" />}
      </div>
      <div className={cn('min-w-0 flex-1', !isLast && (compact ? 'pb-3' : 'pb-8'))}>
        <h3
          className={cn(
            'text-basis font-medium',
            compact ? 'text-xs leading-5' : 'mb-2 text-base leading-7'
          )}
        >
          {title}
        </h3>
        {compact && children ? <div className="mt-1.5">{children}</div> : children}
      </div>
    </div>
  );
}
