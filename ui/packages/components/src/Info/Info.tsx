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
      className="border-subtle text-subtle mt-6 flex flex-col rounded-md border p-0 text-sm leading-tight"
    >
      <div className="border-subtle text-subtle border-b p-3 text-sm leading-tight">{text}</div>

      <div className="p-3">{action}</div>
    </TooltipContent>
  </Tooltip>
);
