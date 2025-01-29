'use client';

import { Skeleton } from '@inngest/components/Skeleton';
import { cn } from '@inngest/components/utils/classNames';

export const cardWrapperStyles =
  'md:flex w-full lg:min-w-[800px] max-w-[1200px] overflow-hidden rounded-md border border-subtle bg-canvasBase';
const cardLeftPanelStyles =
  'h-36 md:h-56 bg-canvasSubtle flex md:w-[410px] flex-col justify-center gap-2 px-10';
const cardRightPanelStyles = 'h-56 flex flex-col justify-center px-8';

export function SkeletonCard() {
  return (
    <div className={cardWrapperStyles}>
      <div className={cardLeftPanelStyles} />
      <div className={cn(cardRightPanelStyles, 'flex-1')}>
        <Skeleton className="mb-2 block h-8 w-full" />
        <Skeleton className="mb-2 block h-8 w-full" />
        <Skeleton className="mb-2 block h-8 w-full" />
        <Skeleton className="mb-2 block h-8 w-full" />
      </div>
    </div>
  );
}
