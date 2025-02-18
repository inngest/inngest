import { Skeleton } from '@inngest/components/Skeleton';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiInformationLine } from '@remixicon/react';

export default function DescriptionListItem({
  detail,
  term,
  loading = false,
  tooltipContent,
  className,
}: {
  detail: React.ReactNode;
  term: string;
  loading?: boolean;
  tooltipContent?: string | React.ReactNode;
  className?: string;
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
