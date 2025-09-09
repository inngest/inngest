'use client';

import React from 'react';
import { cn } from '@inngest/components/utils/classNames';

// Text cell specifically for Insights results.
// - Wraps text up to a max height, then enables vertical scrolling
// - Keeps horizontal overflow hidden (no horizontal scrolling)
// - Preserves the table's default padding by scoping styles inside the cell
export function InsightsTextCell({ children }: React.PropsWithChildren) {
  return (
    <div
      className={cn(
        'max-h-[100px] w-full min-w-0 overflow-y-auto overflow-x-hidden py-2.5 pr-2',
        // Thin, rounded scrollbar
        '[scrollbar-width:thin]',
        '[&::-webkit-scrollbar]:w-1',
        '[&::-webkit-scrollbar-thumb]:bg-border-subtle',
        '[&::-webkit-scrollbar-thumb]:rounded'
      )}
    >
      <p className="text-basis m-0 w-full text-clip whitespace-pre-wrap break-words text-sm font-medium">
        {children}
      </p>
    </div>
  );
}
