'use client';

import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiFileCopyLine, RiTimeLine, RiUserSmileLine } from '@remixicon/react';
import { toast } from 'sonner';

import { cn } from './utils/classNames';
import { format, relativeTime, toMaybeDate, toUTCDate } from './utils/date';

/**
 * Use this component instead of the builtin <time> element. Since server-side
 * render will use UTC as the locale, we need this component to force a
 * client-side render.
 */

type Props = {
  className?: string;
  format?: 'relative';
  value: Date | string;
};

function formatDate(date: Date) {
  return format(date, 'dd MMM, HH:mm');
}

export function Time({ className, format, value }: Props) {
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success('Copied to clipboard');
  };

  let date: Date | null;
  if (value instanceof Date) {
    date = value;
  } else {
    date = toMaybeDate(value);
  }

  if (!date) {
    return <span>Invalid date</span>;
  }

  let dateString: string;
  if (format === 'relative') {
    dateString = relativeTime(date);
  } else {
    dateString = date.toLocaleString();
  }

  const utcTimeString = formatDate(toUTCDate(date));
  const localTimeString = formatDate(date);

  return (
    <Tooltip>
      <TooltipTrigger>
        <time
          suppressHydrationWarning={true}
          className={cn(
            'hover:bg-canvasSubtle group flex items-center gap-1 whitespace-nowrap pr-4 hover:pr-0',
            className
          )}
          dateTime={date.toISOString()}
          onClick={() => copyToClipboard(date.toISOString())}
        >
          {dateString}
          <RiFileCopyLine className="text-subtle hidden h-3 w-3 group-hover:block" />
        </time>
      </TooltipTrigger>
      <TooltipContent side="bottom" className="w-60 max-w-60 px-0">
        <div className="text-subtle border-subtle border-b px-3 py-2 text-xs">
          Click to copy ISO timestamp
        </div>
        <div className="mb-[6px] ml-3 mr-4 mt-3 flex items-center justify-between gap-2 text-sm">
          <div className="text-muted flex items-center gap-1">
            <RiTimeLine className="h-[14px] w-[14px]" /> UTC
          </div>
          <time className="text-subtle">{utcTimeString}</time>
        </div>
        <div className="mb-[6px] ml-3 mr-4 mt-3 flex items-center justify-between gap-5 text-sm">
          <div className="text-muted flex items-center gap-1">
            <RiUserSmileLine className="h-[14px] w-[14px]" /> Your timezone
          </div>
          <time className="text-subtle">{localTimeString}</time>
        </div>
      </TooltipContent>
    </Tooltip>
  );
}
