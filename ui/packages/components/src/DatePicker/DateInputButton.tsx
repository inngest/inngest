import { forwardRef } from 'react';
import { RiCalendarLine } from '@remixicon/react';

export type DateInputButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement>;

export const DateInputButton = forwardRef<HTMLButtonElement, DateInputButtonProps>(
  ({ children, className, ...props }, forwardRef) => {
    return (
      <button
        {...props}
        ref={forwardRef}
        className={`border-muted h-8 rounded-lg border bg-white px-3.5 text-sm leading-none shadow outline-2 outline-indigo-500 transition-all focus:outline ${className}`}
      >
        <span className="flex items-center gap-2">
          <RiCalendarLine className="text-disabled h-5 w-5" />
          {children}
        </span>
      </button>
    );
  }
);
