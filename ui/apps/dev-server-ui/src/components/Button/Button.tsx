import React, { forwardRef } from 'react';

import { IconSpinner } from '@/icons';
import classNames from '@/utils/classnames';
import {
  getButtonColors,
  getButtonSizeStyles,
  getDisabledStyles,
  getIconSizeStyles,
  getKeyColor,
} from './buttonStyles';

interface ButtonProps {
  kind?: 'default' | 'primary' | 'success' | 'danger';
  appearance?: 'solid' | 'outlined' | 'text';
  size?: 'small' | 'regular' | 'large';
  label?: React.ReactNode;
  icon?: React.ReactNode;
  disabled?: boolean;
  loading?: boolean;
  type?: 'submit' | 'button';
  btnAction?: (e?: React.MouseEvent) => void;
  keys?: string[];
  isSplit?: boolean;
  className?: string;
}

export default forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    kind = 'default',
    appearance = 'solid',
    size = 'small',
    label,
    icon,
    loading = false,
    disabled,
    btnAction,
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
  const disabledStyles = getDisabledStyles();
  const iconSizes = getIconSizeStyles({ size });
  const keyColor = getKeyColor({ kind, appearance });

  const iconElement = React.isValidElement(icon)
    ? React.cloneElement(icon as React.ReactElement, {
        className: !label
          ? classNames('h-4 w-4', icon.props.className)
          : classNames(iconSizes, icon.props.className),
      })
    : null;

  return (
    <button
      ref={ref}
      className={classNames(
        buttonColors,
        buttonSizes,
        disabledStyles,
        isSplit ? 'rounded-l' : 'rounded',
        'flex gap-1.5 items-center justify-center drop-shadow-sm transition-all active:scale-95 ',
        className
      )}
      type={type}
      onClick={btnAction}
      disabled={disabled}
      {...props}
    >
      {loading && <IconSpinner className={`fill-white ${iconSizes}`} />}
      {!loading && iconElement}
      {label && label}
      {!loading && keys && (
        <kbd className="ml-auto flex items-center gap-1">
          {keys.map((key, i) => (
            <kbd
              className={classNames(
                keyColor,
                'ml-auto flex h-6 w-6 items-center justify-center rounded font-sans text-xs'
              )}
            >
              {key}
            </kbd>
          ))}
        </kbd>
      )}
    </button>
  );
});
