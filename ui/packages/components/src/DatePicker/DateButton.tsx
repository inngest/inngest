import { forwardRef } from 'react';
import { RiArrowDownSLine, RiCalendarLine } from '@remixicon/react';

import { cn } from '../utils/classNames';

export type DateButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement>;

export const DateInputButton = forwardRef<HTMLButtonElement, DateButtonProps>(
  ({ children, className, ...props }, forwardRef) => {
    return (
      <button
        {...props}
        ref={forwardRef}
        className={`border-muted bg-canvasBase outline-primary-moderate h-8 rounded-md border px-2 text-sm leading-none outline-2 transition-all focus:outline ${className}`}
      >
        <span className="flex items-center gap-2">
          <RiCalendarLine className="text-disabled h-5 w-5" />
          {children}
        </span>
      </button>
    );
  }
);

export const DateSelectButton = forwardRef<HTMLButtonElement, DateButtonProps>(
  ({ children, className, ...props }, forwardRef) => {
    return (
      <button
        {...props}
        ref={forwardRef}
        className={cn(
          'border-muted bg-surfaceBase text-basis group box-content flex h-[38px] w-full items-center justify-between rounded-l-[5px] rounded-r-[5px] border px-2 text-sm',
          className
        )}
      >
        {children}
        <RiArrowDownSLine
          className="text-muted h-4 w-4 transition-transform duration-500 group-data-[state=open]:-rotate-180"
          aria-hidden="true"
        />
      </button>
    );
  }
);
