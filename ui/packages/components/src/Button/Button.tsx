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

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  kind?: ButtonKind;
  appearance?: ButtonAppearance;
  size?: ButtonSize;
  loading?: boolean;
  href?: string | UrlObject;
  target?: string;
  tooltip?: ReactNode;
  label?: ReactNode;
  icon?: ReactNode;
  iconSide?: 'right' | 'left';
  keys?: string[];
  prefetch?: boolean;
  scroll?: boolean;
}

type LinkWrapperProps = {
  children: ReactNode;
  href?: string | UrlObject;
  target?: string;
  prefetch?: boolean;
  scroll?: boolean;
} & Omit<NextLinkProps, 'href'>;

export const LinkWrapper = forwardRef<HTMLAnchorElement, LinkWrapperProps>(
  ({ children, href, target, prefetch = false, scroll = true, ...props }, ref) =>
    href ? (
      <NextLink
        href={href as Route}
        target={target}
        prefetch={prefetch}
        scroll={scroll}
        ref={ref}
        {...props}
      >
        {children}
      </NextLink>
    ) : (
      children
    )
);
LinkWrapper.displayName = 'LinkWrapper';

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

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      kind = 'primary',
      appearance = 'solid',
      size = 'medium',
      label,
      icon,
      iconSide,
      loading = false,
      href,
      type = 'button',
      keys,
      className,
      tooltip,
      disabled,
      target,
      prefetch = false,
      scroll = true,
      ...props
    }: ButtonProps,
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
        {icon && iconSide === 'left' && (
          <span className={cn(size === 'small' ? 'pr-1' : 'pr-1.5')}>{iconElement}</span>
        )}
        {label ? (
          <span className={loading ? 'invisible' : 'visible'}>{label}</span>
        ) : (
          icon && !iconSide && iconElement
        )}
        {icon && iconSide === 'right' && (
          <span className={cn(size === 'small' ? 'pl-1' : 'pl-1.5')}>{iconElement}</span>
        )}
      </>
    );

    return (
      <TooltipWrapper tooltip={tooltip}>
        <LinkWrapper href={href} target={target} prefetch={prefetch} scroll={scroll}>
          <button
            ref={ref}
            className={cn(
              buttonColors,
              buttonSizes,
              'flex items-center justify-center whitespace-nowrap rounded-md disabled:cursor-not-allowed',
              className
            )}
            type={type}
            disabled={disabled}
            {...props}
          >
            {children}
          </button>
        </LinkWrapper>
      </TooltipWrapper>
    );
  }
);
