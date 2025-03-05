import type { UrlObject } from 'url';
import React, { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react';
import type { Route } from 'next';
import NextLink, { type LinkProps as NextLinkProps } from 'next/link';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { IconSpinner } from '@inngest/components/icons/Spinner';

import { cn } from '../utils/classNames';
import {
  getButtonColors,
  getButtonSizeStyles,
  getIconSizeStyles,
  getSpinnerStyles,
} from './buttonStyles';

export type ButtonKind = 'primary' | 'secondary' | 'danger';
export type ButtonAppearance = 'solid' | 'outlined' | 'ghost';
export type ButtonSize = 'small' | 'medium' | 'large';

export interface SplitButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  kind?: ButtonKind;
  appearance?: ButtonAppearance;
  size?: ButtonSize;
  loading?: boolean;
  tooltip?: ReactNode;
  label?: ReactNode;
  icon?: ReactNode;
}

export const TooltipWrapper = ({
  children,
  tooltip,
}: {
  children: ReactNode;
  tooltip?: ReactNode;
}) =>
  tooltip ? (
    <Tooltip>
      <TooltipTrigger asChild>{children}</TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  ) : (
    children
  );

export const SplitButton = forwardRef<HTMLButtonElement, SplitButtonProps>(
  (
    {
      kind = 'primary',
      appearance = 'solid',
      size = 'medium',
      label,
      icon,
      loading = false,
      type = 'button',
      className,
      tooltip,
      disabled,
      ...props
    }: SplitButtonProps,
    ref
  ) => {
    const buttonColors = getButtonColors({ kind, appearance, loading });
    const buttonSizes = getButtonSizeStyles({ size, icon, label });
    const spinnerStyles = getSpinnerStyles({ kind, appearance });
    const iconSizes = getIconSizeStyles({ size });

    const iconElement = React.isValidElement(icon)
      ? React.cloneElement(icon as React.ReactElement, {
          className: cn(iconSizes, icon.props.className, loading && 'invisible'),
        })
      : null;

    const children = (
      <>
        {loading && (
          <IconSpinner className={cn(spinnerStyles, iconSizes, 'top-50% left-50% absolute')} />
        )}

        {label && <span className={loading ? 'invisible' : 'visible'}>{label}</span>}
      </>
    );

    return (
      <TooltipWrapper tooltip={tooltip}>
        <div className="flex flex-row items-center justify-center">
          <button
            ref={ref}
            className={cn(
              buttonColors,
              buttonSizes,
              'mr-0 flex items-center justify-end whitespace-nowrap rounded-md rounded-r-none disabled:cursor-not-allowed',
              className
            )}
            type={type}
            disabled={disabled}
            {...props}
          >
            {children}
          </button>
          <button
            ref={ref}
            className={cn(
              buttonColors,
              buttonSizes,
              'ml-0 flex items-center justify-end whitespace-nowrap rounded-md rounded-l-none border-l-0 px-1 py-1.5 disabled:cursor-not-allowed',
              className
            )}
            type={type}
            disabled={disabled}
            {...props}
          >
            <span className={cn(size === 'small' ? '' : '')}>{iconElement}</span>
          </button>
        </div>
      </TooltipWrapper>
    );
  }
);
