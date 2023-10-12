import type { UrlObject } from 'url';
import type { Route } from 'next';
import Link, { type LinkProps } from 'next/link';

import cn from '@/utils/cn';

const primaryStyles =
  'bg-gradient-to-b from-[#6d7bfe] to-[#6366f1] hover:from-[#7986fd] hover:to-[#7679f9] text-shadow text-white';

const contextStyles = {
  'danger-light':
    'shadow-outline-secondary-light bg-white text-slate-700 hover:bg-slate-100 hover:text-red-700 font-medium text-red-500',
  'danger-dark':
    'shadow-outline-secondary-dark bg-slate-800 text-red-400 hover:bg-slate-700 hover:text-red-200',
  'primary-light': `${primaryStyles} font-medium`,
  'primary-dark': `shadow-outline-primary-dark ${primaryStyles} font-medium`,
  'secondary-light': `shadow-outline-secondary-light bg-white text-slate-700 hover:bg-slate-100 hover:text-indigo-500 font-medium`,
  'secondary-dark': `shadow-outline-secondary-dark bg-slate-800 text-white hover:bg-slate-700`,
  'text-light': 'text-indigo-600 hover:text-indigo-900 hover:underline',
  'text-dark': 'text-indigo-300 hover:text-indigo-200 hover:underline',
  'text-danger-light': 'text-red-500 hover:text-red-700 hover:underline',
  'text-danger-dark': 'text-red-400 hover:text-red-200 hover:underline',
} as const;

const sizeStyles = {
  sm: 'px-2.5 py-1 h-[26px] text-xs rounded',
  base: 'px-3 h-8',
  lg: 'px-6 py-2.5',
};

const disabledStyles = {
  danger: 'opacity-50 cursor-not-allowed hover:text-inherit',
  primary: 'opacity-50 cursor-not-allowed hover:bg-inherit hover:no-underline',
  secondary: 'opacity-50 cursor-not-allowed hover:bg-inherit hover:no-underline',
  text: 'opacity-50 cursor-not-allowed hover:text-inherit',
  'text-danger': 'opacity-50 cursor-not-allowed hover:text-inherit',
} as const;

type ButtonProps<PassedHref extends string> = {
  variant?: 'danger' | 'primary' | 'secondary' | 'text' | 'text-danger';
  context?: 'dark' | 'light';
  type?: 'button' | 'submit';
  size?: 'base' | 'sm' | 'lg';
  href?: Route<PassedHref> | UrlObject;
  className?: string;
  iconSide?: 'left' | 'right';
  icon?: React.ReactNode;
  disabled?: boolean;
  target?: string;
  rel?: string;
  children?: React.ReactNode;
} & Omit<LinkProps<PassedHref>, 'href'> &
  React.ComponentProps<'button'>;

export default function Button<PassedHref extends string>({
  variant = 'primary',
  context = 'light',
  type = 'button',
  size = 'base',
  iconSide = 'left',
  href,
  icon,
  className,
  children,
  ...props
}: ButtonProps<PassedHref>): JSX.Element {
  const classNames = cn(
    'inline-flex flex-shrink-0 leading-none items-center gap-1 justify-center overflow-hidden text-sm font-regular transition rounded-[6px] transition-all',
    contextStyles[`${variant}-${context}` as const],
    sizeStyles[size],
    props.disabled && disabledStyles[variant],
    className
  );

  if (href) {
    return (
      <Link className={classNames} href={href} {...props} type={type}>
        {iconSide === 'left' && icon}
        {children}
        {iconSide === 'right' && icon}
      </Link>
    );
  }

  return (
    <button className={classNames} {...props} type={type}>
      {iconSide === 'left' && icon}
      {children}
      {iconSide === 'right' && icon}
    </button>
  );
}
