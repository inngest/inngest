import { forwardRef, type InputHTMLAttributes } from 'react';

import { cn } from '../utils/classNames';

export type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  allowPasswordManager?: boolean;
  label?: string;
  error?: string | undefined;
  inngestSize?: 'base' | 'lg';
  className?: string;
  optional?: boolean;
};

const sizeStyles = {
  base: 'text-sm px-2 py-2 h-8',
  lg: 'text-sm px-3.5 py-3',
};

export const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    { allowPasswordManager = false, className, type = 'text', inngestSize = 'base', ...props },
    ref
  ) => {
    let passwordManagerProps: Record<string, unknown> = {
      autoComplete: 'off',
      'data-1p-ignore': true,
      'data-bwignore': true,
      'data-lpignore': true,
    };
    if (allowPasswordManager) {
      passwordManagerProps = {};
    }

    return (
      <div className="flex flex-col gap-1">
        {props.label && (
          <label htmlFor={props.name} className="text-basis text-sm font-medium">
            {props.label}{' '}
            {props.optional && <span className="text-subtle font-normal">(optional)</span>}
          </label>
        )}
        <div className="flex">
          <input
            ref={ref}
            className={cn(`bg-canvasBase border-muted placeholder-disabled text-basis outline-primary-moderate w-full rounded-md border text-sm leading-none outline-2 transition-all focus:outline
            ${sizeStyles[inngestSize]}
            ${
              props.readOnly &&
              'bg-disabled text-disabled cursor-not-allowed border-transparent	shadow-transparent outline-transparent'
            }
            ${props.error && 'outline-error'}
            ${className}`)}
            {...passwordManagerProps}
            {...props}
          />
        </div>

        {props.error && <p className="text-error text-sm">{props.error}</p>}
      </div>
    );
  }
);
