import Link from 'next/link';

import cn from '@/utils/cn';

const kindStyles = {
  success: 'text-teal-500',
  warn: 'text-orange-500',
  danger: 'text-red-500',
};

const sizeStyles = {
  sm: 'text-xs px-2',
  base: 'text-sm py-1.5 px-3',
};

export function StatusTag({
  children,
  className = '',
  kind = 'success',
  size = 'sm',
}: {
  children: React.ReactNode;
  className?: string;
  kind?: 'success' | 'warn' | 'danger';
  size?: 'sm' | 'base';
}) {
  const classNames = cn(
    'rounded-[6px] inline-flex border items-center leading-none font-semibold border-slate-800 bg-slate-700 ',
    kindStyles[kind],
    sizeStyles[size],
    className
  );

  return <span className={classNames}>{children}</span>;
}
