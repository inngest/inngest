'use client';

import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { RiCheckLine, RiFileCopy2Line } from '@remixicon/react';

type Props = {
  value: string;
};

export function CopyButton({ value }: Props) {
  const { handleCopyClick, isCopying } = useCopyToClipboard();

  let Icon = RiFileCopy2Line;
  if (isCopying) {
    Icon = RiCheckLine;
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
