import { forwardRef, type HTMLAttributes } from 'react';

import cn from '@/utils/cn';

type InputProps = {
  defaultValue?: HTMLAttributes<HTMLInputElement>['defaultValue'];
  error?: string;
  showError?: boolean;
  name?: string;
  id?: string;
  label?: string;
  placeholder?: string;
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  type?: 'text' | 'password' | 'email' | 'number';
  size?: 'base' | 'lg';
  className?: string;
  value?: string;
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  onFocus?: (e: React.FocusEvent<HTMLInputElement>) => void;
  onBlur?: (e: React.FocusEvent<HTMLInputElement>) => void;
  readonly?: boolean;
};

const sizeStyles = {
  base: 'text-sm px-2 py-2 h-8 rounded-[6px]',
  lg: 'text-sm px-3.5 py-3 rounded-lg',
};

/**
 * @deprecated Use shared Input component instead
 */
const Input = forwardRef<HTMLInputElement, InputProps>(({ showError = true, ...props }, ref) => {
  const type = props.type === undefined ? 'text' : props.type;
  const size = props.size === undefined ? 'base' : props.size;
  const placeholder = props.placeholder === undefined ? '' : props.placeholder;
  const className = props.className === undefined ? '' : props.className;
  return (
    <div className="flex flex-col">
      {props.label && (
        <label htmlFor={props.name} className="text-basis text-sm font-medium">
          {props.label}
        </label>
      )}
      <input
        ref={ref}
        defaultValue={props.defaultValue}
        required={props.required}
        minLength={props.minLength}
        maxLength={props.maxLength}
        type={type}
        name={props.name}
        id={props.id}
        placeholder={placeholder}
        value={props.value}
        className={cn(
          'border-muted placeholder-disabled outline-primary-moderate border text-sm leading-none outline-2 transition-all focus:outline',
          sizeStyles[size],
          props.readonly && 'cursor-not-allowed border-transparent outline-transparent	',
          props.error && 'border-error',
          className
        )}
        onChange={props.onChange}
        onKeyDown={props.onKeyDown}
        onFocus={props.onFocus}
        onBlur={props.onBlur}
        readOnly={props.readonly}
      />

      <p className="text-error text-sm">{showError && props.error}</p>
    </div>
  );
});

Input.displayName = 'Input';

export default Input;
