'use client';

import type { ReactNode } from 'react';
import { RiQuestionLine } from '@remixicon/react';

import { Popover, PopoverContent, PopoverTrigger } from '../Popover';

export const Info = ({ text, action }: { text: string | ReactNode; action: ReactNode }) => (
  <Popover>
    <PopoverTrigger>
      <RiQuestionLine className="text-subtle h-[18px] w-[18px]" />
    </PopoverTrigger>
    <PopoverContent
      side="right"
      align="start"
      className="text-subtle flex max-w-xs flex-col text-sm leading-tight"
    >
      <div className="border-subtle border-b px-4 py-2 text-sm leading-tight">{text}</div>

      <div className="px-4 py-2">{action}</div>
    </PopoverContent>
  </Popover>
);
