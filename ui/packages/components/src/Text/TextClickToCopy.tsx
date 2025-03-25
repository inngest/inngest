'use client';

import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiClipboardLine } from '@remixicon/react';
import { toast } from 'sonner';

export function TextClickToCopy({
  truncate = false,
  children,
}: {
  truncate?: boolean;
  children: string;
}) {
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success('Copied to clipboard');
  };
  return (
    <span className="flex">
      <Tooltip>
        <TooltipTrigger
          className={cn('block', truncate && 'truncate')}
          onClick={() => {
            copyToClipboard(children);
          }}
        >
          <span className={cn('block', truncate && 'truncate')}>{children}</span>
        </TooltipTrigger>
        <TooltipContent className="flex items-center gap-1">
          <RiClipboardLine className="h-3 w-3" /> Click to copy
        </TooltipContent>
      </Tooltip>
    </span>
  );
}
