import ProgressBar from '@inngest/components/ProgressBar/ProgressBar';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiInformationLine } from '@remixicon/react';

export type Data = {
  title: string;
  description: string;
  current: number;
  limit: number | null;
  overageAllowed?: boolean;
  tooltipContent?: string | React.ReactNode;
};

export async function LimitBar({ data, className }: { data: Data; className?: string }) {
  const { title, description, current, limit, overageAllowed, tooltipContent } = data;
  const isUnlimited = limit === null;
  return (
    <div className={cn(className)}>
      <div className="text-subtle mb-1 flex items-center gap-1 text-sm font-medium">
        {title}
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
      </div>
      <p className="text-muted mb-2 text-sm italic">{description}</p>
      <ProgressBar value={current} limit={limit} overageAllowed={overageAllowed} />
      <div className="mt-1 text-left">
        <span
          className={cn(
            'text-medium text-basis text-sm font-medium',
            !isUnlimited && current > limit && overageAllowed && 'text-warning',
            !isUnlimited && current > limit && !overageAllowed && 'text-error'
          )}
        >
          {current.toLocaleString()}
        </span>
        <span className="text-muted text-sm">
          /{isUnlimited ? 'unlimited' : limit.toLocaleString()}
        </span>
      </div>
    </div>
  );
}
