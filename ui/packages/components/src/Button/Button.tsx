import type { UrlObject } from 'url';
import React, { forwardRef } from 'react';
import Link from 'next/link';
import { IconSpinner } from '@inngest/components/icons/Spinner';

import { classNames } from '../utils/classNames';
import {
  getButtonColors,
  getButtonSizeStyles,
  getDisabledStyles,
  getIconSizeStyles,
  getKeyColor,
  getSpinnerStyles,
} from './buttonStyles';

interface ButtonProps {
  kind?: 'default' | 'primary' | 'success' | 'danger';
  appearance?: 'solid' | 'outlined' | 'text';
  size?: 'small' | 'regular' | 'large';
  iconSide?: 'right' | 'left';
  label?: React.ReactNode;
  icon?: React.ReactNode;
  disabled?: boolean;
  loading?: boolean;
  type?: 'submit' | 'button';
  btnAction?: (e?: React.MouseEvent) => void;
  href?: string | UrlObject;
  keys?: string[];
  isSplit?: boolean;
  className?: string;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    kind = 'default',
    appearance = 'solid',
    size = 'small',
    label,
    icon,
    iconSide = 'left',
    loading = false,
    disabled,
    btnAction,
    href,
    isSplit,
    type,
    keys,
    className,
    ...props
  }: ButtonProps,
  ref
) {
  const buttonColors = getButtonColors({ kind, appearance });
  const buttonSizes = getButtonSizeStyles({ size, icon, label });
  const disabledStyles = getDisabledStyles({ kind, appearance });
  const spinnerStyles = getSpinnerStyles({ kind, appearance });
  const iconSizes = getIconSizeStyles({ size });
  const keyColor = getKeyColor({ kind, appearance });

  const iconElement = React.isValidElement(icon)
    ? React.cloneElement(icon as React.ReactElement, {
        className: !label
          ? classNames('h-4 w-4', icon.props.className)
          : classNames(iconSizes, icon.props.className),
      })
    : null;

  const children = (
    <>
      {loading && <IconSpinner className={classNames(spinnerStyles, iconSizes)} />}
      {!loading && iconSide === 'left' && iconElement}
      {label && label}
      {!loading && iconSide === 'right' && iconElement}
      {!loading && keys && (
        <kbd className="ml-auto flex items-center gap-1">
          {keys.map((key, i) => (
            <kbd
              key={i}
              className={classNames(
                disabled
                  ? 'bg-slate-200 text-slate-400 dark:bg-slate-800 dark:text-slate-500'
                  : keyColor,
                'ml-auto flex h-6 w-6 items-center justify-center rounded font-sans text-xs'
              )}
            >
              {key}
            </kbd>
          ))}
        </kbd>
      )}
    </>
  );

  const Element = href ? (
    <Link
      className={classNames(
        buttonColors,
        buttonSizes,
        disabledStyles,
        'flex items-center justify-center gap-1.5 drop-shadow-sm transition-all active:scale-95 ',
        className
      )}
      href={href}
      {...props}
    >
      {children}
    </Link>
  ) : (
    <button
      ref={ref}
      className={classNames(
        buttonColors,
        buttonSizes,
        disabledStyles,
        isSplit ? 'rounded-l' : 'rounded',
        'flex items-center justify-center gap-1.5 drop-shadow-sm transition-all active:scale-95 ',
        className
      )}
      type={type}
      onClick={btnAction}
      disabled={disabled}
      {...props}
    >
      {children}
    </button>
  );

  return Element;
});
