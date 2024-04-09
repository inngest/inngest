import { forwardRef } from 'react';
import CalendarIcon from '@heroicons/react/20/solid/CalendarIcon';

type DateInputButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement>;

export const DateInputButton = forwardRef<HTMLButtonElement, DateInputButtonProps>(
  ({ children, ...props }, forwardRef) => (
    <button
      {...props}
      ref={forwardRef}
      className="h-8 rounded-lg border border-slate-300 bg-white px-3.5 text-sm leading-none placeholder-slate-500 shadow outline-2 outline-indigo-500 transition-all focus:outline"
    >
      <span className="flex items-center gap-2">
        <CalendarIcon className="h-6 w-6" />
        {children}
      </span>
    </button>
  )
);
