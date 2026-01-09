import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiFileCopyLine, RiTimeLine, RiUserSmileLine } from '@remixicon/react';
import { toast } from 'sonner';

import { cn } from './utils/classNames';
import { format, formatInTimeZone, relativeTime, toMaybeDate } from './utils/date';

/**
 * Use this component instead of the builtin <time> element. Since server-side
 * render will use UTC as the locale, we need this component to force a
 * client-side render.
 */

type Props = {
  className?: string;
  format?: 'relative';
  value: Date | string;
  copyable?: boolean;
};

function formatDate(date: Date) {
  return format(date, 'dd MMM yyyy, hh:mm:ss a');
}

function formatUTCDate(date: Date) {
  return formatInTimeZone(date, 'UTC', 'dd MMM yyyy, HH:mm:ss');
}

export function Time({ className, format, value, copyable = true }: Props) {
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success('Copied to clipboard');
  };

  const date = value instanceof Date ? value : toMaybeDate(value);

  if (!(date instanceof Date) || isNaN(date)) {
    return <span>Invalid date</span>;
  }

  const dateString = format === 'relative' ? relativeTime(date) : date.toLocaleString();

  const utcTimeString = formatUTCDate(date);
  const localTimeString = formatDate(date);

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <time
          suppressHydrationWarning={true}
          className={cn(
            'group flex items-center gap-1 whitespace-nowrap',
            copyable && 'hover:bg-canvasSubtle',
            className
          )}
          dateTime={date.toISOString()}
        >
          {dateString}
          {copyable && (
            <RiFileCopyLine
              className="text-subtle h-4 w-4 cursor-pointer opacity-0 transition-opacity group-hover:opacity-100"
              onClick={(e) => {
                e.stopPropagation();
                e.preventDefault();
                copyToClipboard(date.toISOString());
              }}
            />
          )}
        </time>
      </TooltipTrigger>
      <TooltipContent side="right" className="w-auto px-0">
        <div className="mb-[6px] ml-3 mr-4 mt-1.5 flex items-center gap-2 whitespace-nowrap text-sm">
          <div className="text-light flex w-16 items-center gap-1">
            <RiTimeLine className="h-[14px] w-[14px]" /> UTC
          </div>
          <time className="text-onContrast">{utcTimeString}</time>
        </div>
        <div className="mb-[6px] ml-3 mr-4 flex items-center gap-2 whitespace-nowrap text-sm">
          <div className="text-light flex w-16 items-center gap-1">
            <RiUserSmileLine className="h-[14px] w-[14px]" /> Local
          </div>
          <time className="text-onContrast">{localTimeString}</time>
        </div>
      </TooltipContent>
    </Tooltip>
  );
}
