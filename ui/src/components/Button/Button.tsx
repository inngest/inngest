import React from 'react';

import { IconSpinner } from '@/icons';
import classNames from '@/utils/classnames';

interface ButtonProps {
  kind?: 'default' | 'primary' | 'success' | 'danger';
  appearance?: 'solid' | 'outlined';
  size?: 'small' | 'regular' | 'large';
  label?: React.ReactNode;
  icon?: React.ReactNode;
  disabled?: boolean;
  loading?: boolean;
  type?: 'submit' | 'button';
  btnAction?: (e?: React.MouseEvent) => void;
  keys?: string[];
}

const kindColors = {
  default: 'slate-700',
  primary: 'indigo-500',
  success: 'emerald-600',
  danger: 'red-700',
};

const sizeStyles = {
  small: 'text-xs px-2.5 h-7',
  regular: 'text-sm px-2.5 h-8',
  large: 'text-base px-2.5 h-10',
};

const iconOnlySizeStyles = {
  small: 'text-xs w-7 h-7',
  regular: 'text-sm w-8 h-8',
  large: 'text-base w-10 h-10',
};

export default function Button({
  kind = 'default',
  appearance = 'solid',
  size = 'small',
  label,
  icon,
  loading = false,
  disabled,
  btnAction,

  type,
  keys,
}: ButtonProps) {
  const buttonColors =
    appearance === 'solid'
      ? `bg-${kindColors[kind]} border-t border-white/10 hover:bg-${kindColors[kind]}/80 text-slate-100`
      : `bg-${kindColors[kind]}/20 border border-${kindColors[kind]}/80 hover:border-${kindColors[kind]} text-slate-200`;
  const buttonSizes = icon && !label ? iconOnlySizeStyles[size] : sizeStyles[size];
  const keyColor =
    appearance === 'solid' && kind === 'default'
      ? `bg-slate-800`
      : appearance === 'solid'
      ? `bg-slate-800/20`
      : `bg-${kindColors[kind]}/80`;

  const disabledStyles =
    'disabled:text-slate-500 disabled:cursor-not-allowed disabled:bg-slate-800 disabled:hover:bg-slate-800 disabled:border-slate-800';

  // Replace this with alternative once we revamp the button variations
  const iconElement = icon
    ? React.cloneElement(icon as React.ReactElement, { className: 'icon-xs' })
    : null;

  return (
    <button
      className={classNames(
        buttonColors,
        buttonSizes,
        disabledStyles,
        'flex gap-1.5 items-center justify-center rounded-sm drop-shadow-sm',
      )}
      type={type}
      onClick={btnAction}
      disabled={disabled}
    >
      {loading && <IconSpinner className="fill-white icon-xs" />}
      {!loading && iconElement}
      {label && label}
      {keys && (
        <kbd className="ml-auto flex items-center gap-1">
          {keys.map((key, i) => (
            <kbd
              className={classNames(
                keyColor,
                'ml-auto flex h-6 w-6 items-center justify-center rounded-sm font-sans text-xs',
              )}
            >
              {key}
            </kbd>
          ))}
        </kbd>
      )}
    </button>
  );
}
