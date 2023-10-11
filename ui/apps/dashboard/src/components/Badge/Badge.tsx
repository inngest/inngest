import { createElement } from 'react';
import Link from 'next/link';

const kindStyles = {
  solid: 'border-slate-600 bg-slate-600 text-white',
};

const sizeStyles = {
  sm: 'text-xs py-1 px-2 h-[26px]',
  base: 'text-sm py-1.5 px-3',
};

export function Badge({
  children,
  className = '',
  kind = 'solid',
  size = 'base',
}: {
  children: React.ReactNode;
  className?: string;
  kind?: 'solid';
  size?: 'sm' | 'base';
}) {
  const classNames = `rounded-full inline-flex border items-center leading-none font-regular ${kindStyles[kind]} ${sizeStyles[size]} ${className}`;

  return <span className={classNames}>{children}</span>;
}
