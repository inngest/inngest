'use client';

import { Skeleton } from '@inngest/components/Skeleton';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiInformationLine } from '@remixicon/react';

type Props = {
  title: string;
  className?: string;
};

export function AppDetailsCard({ title, className, children }: React.PropsWithChildren<Props>) {
  return (
    <>
      <div className={cn('border-subtle bg-canvasSubtle rounded-md border', className)}>
        <h2 className="text-muted border-subtle border-b px-6 py-3 text-sm">{title}</h2>

        <dl className="bg-canvasBase flex flex-col gap-4 rounded-b-md p-6 md:grid md:grid-cols-4">
          {children}
        </dl>
      </div>
    </>
  );
}

export function CardItem({
  className,
  detail,
  term,
  loading = false,
  tooltipContent,
}: {
  className?: string;
  detail: React.ReactNode;
  term: string;
  loading?: boolean;
  tooltipContent?: string | React.ReactNode;
}) {
  return (
    <div className={className}>
      <dt className="text-light flex items-center gap-1 pb-1 text-sm">
        {term}
        {tooltipContent && (
          <Tooltip>
            <TooltipTrigger>
              <RiInformationLine className="text-light h-4 w-4" />
            </TooltipTrigger>
            <TooltipContent className="whitespace-pre-line text-left">
              {tooltipContent}
            </TooltipContent>
          </Tooltip>
        )}
      </dt>
      {!loading && (
        <dd
          className="text-subtle truncate text-sm"
          title={typeof detail === 'string' ? detail : undefined}
        >
          {detail ?? ''}
        </dd>
      )}
      {loading && <Skeleton className="mb-2 block h-6 w-full" />}
    </div>
  );
}
