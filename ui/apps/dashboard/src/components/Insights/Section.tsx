'use client';

import type { ReactNode } from 'react';

export interface SectionProps {
  children: ReactNode;
  className?: string;
  actions?: ReactNode;
  title: string;
}

export function Section({ actions, children, className = '', title }: SectionProps) {
  return (
    <div className={`flex h-full w-full min-w-0 flex-col ${className}`}>
      <header className="border-subtle flex w-full min-w-0 items-center justify-between border-b px-4 py-3">
        <h2 className="text-basis min-w-0 text-xs font-medium uppercase tracking-wide">{title}</h2>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </header>
      <div className="w-full min-w-0 flex-1 overflow-hidden">{children}</div>
    </div>
  );
}
