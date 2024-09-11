import type { ReactNode } from 'react';
import { RiQuestionLine } from '@remixicon/react';

import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';

export const Info = ({ text, action }: { text: string; action: ReactNode }) => (
  <Tooltip>
    <TooltipTrigger>
      <RiQuestionLine className="text-subtle h-[18px] w-[18px]" />
    </TooltipTrigger>
    <TooltipContent
      side="right"
      sideOffset={2}
      className="border-subtle text-muted text-md mt-6 flex flex-col rounded-lg border p-0"
    >
      <div className="border-subtle border-b px-4 py-2 ">{text}</div>

      <div className="px-4 py-2">{action}</div>
    </TooltipContent>
  </Tooltip>
);
