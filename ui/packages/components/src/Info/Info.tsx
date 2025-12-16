'use client';

import type { ComponentProps, ReactNode } from 'react';
import { cn } from '@inngest/components/utils/classNames';
import { RiQuestionLine } from '@remixicon/react';

import { Popover, PopoverContent, PopoverTrigger } from '../Popover';

export const Info = ({
  text,
  action,
  iconElement,
  widthClassName = 'max-w-xs',
  side = 'right',
  align = 'start',
}: {
  text: string | ReactNode;
  action: ReactNode;
  iconElement?: ReactNode;
  widthClassName?: string;
  side?: ComponentProps<typeof PopoverContent>['side'];
  align?: ComponentProps<typeof PopoverContent>['align'];
}) => {
  const icon = iconElement ?? <RiQuestionLine className="text-subtle h-[18px] w-[18px]" />;

  return (
    <Popover>
      <PopoverTrigger>{icon}</PopoverTrigger>
      <PopoverContent
        side={side}
        align={align}
        className={cn('text-subtle flex flex-col text-sm leading-tight', widthClassName)}
      >
        <div className="border-subtle border-b px-4 py-2 text-sm leading-tight">{text}</div>

        <div className="px-4 py-2">{action}</div>
      </PopoverContent>
    </Popover>
  );
};
