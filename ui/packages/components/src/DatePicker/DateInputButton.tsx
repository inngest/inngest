import { forwardRef } from 'react';
import { RiCalendarLine } from '@remixicon/react';

export type DateInputButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement>;

export const DateInputButton = forwardRef<HTMLButtonElement, DateInputButtonProps>(
  ({ children, className, ...props }, forwardRef) => {
    return (
      <button
        {...props}
        ref={forwardRef}
        className={`h-8 rounded-lg border border-slate-300 bg-white px-3.5 text-sm leading-none shadow outline-2 outline-indigo-500 transition-all focus:outline ${className}`}
      >
        <span className="flex items-center gap-2">
          <RiCalendarLine className="h-6 w-6" />
          {children}
        </span>
      </button>
    );
  }
);
