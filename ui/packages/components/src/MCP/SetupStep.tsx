import type { ReactNode } from 'react';

import { cn } from '../utils/classNames';

export function SetupStep({
  children,
  isLast = false,
  number,
  title,
}: {
  children?: ReactNode;
  isLast?: boolean;
  number: number;
  title: string;
}) {
  return (
    <div className="flex gap-4">
      <div className="flex flex-col items-center">
        <div className="border-subtle bg-canvasBase text-basis flex h-7 w-7 shrink-0 items-center justify-center rounded-full border text-xs font-medium">
          {number}
        </div>
        {!isLast && <div className="border-subtle my-1 w-px flex-1 border-l" />}
      </div>
      <div className={cn('min-w-0 flex-1', !isLast && 'pb-8')}>
        <h3 className="text-basis mb-2 text-base font-medium leading-7">{title}</h3>
        {children}
      </div>
    </div>
  );
}
