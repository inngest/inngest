import { forwardRef, type InputHTMLAttributes } from 'react';

type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  label?: string;
  error?: string | undefined;
  inngestSize?: 'base' | 'lg';
  className?: string;
};

const sizeStyles = {
  base: 'text-sm px-2 py-2 h-8 rounded-[6px]',
  lg: 'text-sm px-3.5 py-3 rounded-lg',
};

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, type = 'text', inngestSize = 'base', ...props }, ref) => {
    return (
      <div className="flex flex-col gap-1">
        {props.label && (
          <label htmlFor={props.name} className="text-sm font-medium text-slate-700">
            {props.label}
          </label>
        )}
        <input
          ref={ref}
          className={`border-muted border text-sm leading-none placeholder-slate-400 shadow outline-2 outline-offset-2 outline-indigo-500 transition-all focus:outline
            ${sizeStyles[inngestSize]}
            ${
              props.readOnly &&
              'cursor-not-allowed border-transparent shadow-transparent outline-transparent	'
            }
            ${props.error && 'outline-red-300'}
            ${className}`}
          {...props}
        />

        {props.error && <p className="text-sm text-red-500">{props.error}</p>}
      </div>
    );
  }
);
