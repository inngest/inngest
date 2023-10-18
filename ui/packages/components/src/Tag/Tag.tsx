import Link from 'next/link';

export function Tag({
  children,
  className = '',
  href,
}: {
  children: React.ReactNode;
  className?: string;
  href?: URL;
}) {
  const classNames = `rounded-full inline-flex items-center h-[26px] px-3 leading-none text-xs font-medium border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 ${className}`;

  if (href) {
    return (
      <Link href={href} className={classNames}>
        {children}
      </Link>
    );
  }

  return <span className={classNames}>{children}</span>;
}
