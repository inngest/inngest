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
      <TooltipTrigger asChild>
        <button
          aria-label={label}
          className="bg-canvasBase flex items-center justify-center px-2"
          onClick={() => handleCopyClick(value)}
        >
          <Icon className="h-6" />
        </button>
      </TooltipTrigger>

      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
