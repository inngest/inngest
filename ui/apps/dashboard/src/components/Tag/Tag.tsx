import Link from 'next/link';

import cn from '@/utils/cn';

const kindStyles = {
  solid: 'border-slate-600 bg-slate-600 text-white',
  subtle: 'border-slate-100 bg-slate-100 text-slate-500',
  outline: 'border-slate-200 text-slate-500',
  'outline-inverse': 'border-slate-800 text-slate-300',
};

const sizeStyles = {
  sm: 'text-xs py-1 px-2 h-[26px]',
  base: 'text-sm py-1.5 px-3',
};

export function Tag({
  children,
  className = '',
  href,
  kind = 'solid',
  size = 'base',
}: {
  children: React.ReactNode;
  className?: string;
  href?: URL;
  kind?: 'solid' | 'subtle' | 'outline' | 'outline-inverse';
  size?: 'sm' | 'base';
}) {
  const classNames = cn(
    'rounded-[6px] inline-flex border items-center leading-none font-regular',
    kindStyles[kind],
    sizeStyles[size],
    className
  );

  if (href) {
    return (
      <Link href={href} className={classNames}>
        {children}
      </Link>
    );
  }

  return <span className={classNames}>{children}</span>;
}
