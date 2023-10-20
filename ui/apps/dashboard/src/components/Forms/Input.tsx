import { type HTMLAttributes } from 'react';

import cn from '@/utils/cn';

type InputProps = {
  defaultValue?: HTMLAttributes<HTMLInputElement>['defaultValue'];
  error?: string | undefined;
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
  readonly?: boolean;
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void;
};

const sizeStyles = {
  base: 'text-sm px-2 py-2 h-8 rounded-[6px]',
  lg: 'text-sm px-3.5 py-3 rounded-lg',
};

export default function Input({
  defaultValue,
  error,
  name,
  id,
  label,
  required,
  minLength,
  maxLength,
  value,
  onChange,
  onKeyDown,
  type = 'text',
  size = 'base',
  placeholder = '',
  className = '',
  readonly,
}: InputProps) {
  return (
    <div className="flex flex-col gap-1">
      {label && (
        <label htmlFor={name} className="text-sm font-medium text-slate-700">
          {label}
        </label>
      )}
      <input
        defaultValue={defaultValue}
        required={required}
        minLength={minLength}
        maxLength={maxLength}
        type={type}
        name={name}
        id={id}
        placeholder={placeholder}
        value={value}
        className={cn(
          'border border-slate-300 text-sm leading-none placeholder-slate-500 shadow outline-2 outline-offset-2 outline-indigo-500 transition-all focus:outline',
          sizeStyles[size],
          readonly && 'border-transparent shadow-transparent outline-transparent',
          className
        )}
        onChange={onChange}
        readOnly={readonly}
        onKeyDown={onKeyDown}
      />

      {error && <p className="text-sm text-red-500">{error}</p>}
    </div>
  );
}
