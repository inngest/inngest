import { forwardRef, type InputHTMLAttributes } from 'react';

type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  allowPasswordManager?: boolean;
  label?: string;
  error?: string | undefined;
  inngestSize?: 'base' | 'lg';
  className?: string;
  optional?: boolean;
};

const sizeStyles = {
  base: 'text-sm px-2 py-2 h-8 rounded-[6px]',
  lg: 'text-sm px-3.5 py-3 rounded-lg',
};

export const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    { allowPasswordManager = false, className, type = 'text', inngestSize = 'base', ...props },
    ref
  ) => {
    let passwordManagerProps: Record<string, unknown> = {
      autocomplete: 'off',
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
        <input
          ref={ref}
          className={`bg-canvasBase border-muted placeholder-disabled text-basis outline-primary-moderate border text-sm leading-none outline-2 transition-all focus:outline
            ${sizeStyles[inngestSize]}
            ${
              props.readOnly &&
              'cursor-not-allowed border-transparent shadow-transparent outline-transparent	'
            }
            ${props.error && 'outline-error'}
            ${className}`}
          {...passwordManagerProps}
          {...props}
        />

        {props.error && <p className="text-error text-sm">{props.error}</p>}
      </div>
    );
  }
);
