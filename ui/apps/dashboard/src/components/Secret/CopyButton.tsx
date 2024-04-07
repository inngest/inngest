'use client';

import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { IconCheck } from '@inngest/components/icons/Check';
import { IconCopy } from '@inngest/components/icons/Copy';

type Props = {
  value: string;
};

export function CopyButton({ value }: Props) {
  const { handleCopyClick, isCopying } = useCopyToClipboard();

  let Icon = IconCopy;
  if (isCopying) {
    Icon = IconCheck;
  }

  const label = 'Copy';

  return (
    <Tooltip>
      <TooltipTrigger className="rounded-r-md bg-white">
        <button
          aria-label={label}
          className="flex w-8 items-center justify-center"
          onClick={() => handleCopyClick(value)}
        >
          <Icon className="h-6" />
        </button>
      </TooltipTrigger>

      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
