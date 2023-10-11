import cn from '@/utils/cn';

export default function PlanBadge({
  className = '',
  variant = 'default',
  children,
}: {
  className?: string;
  variant: 'primary' | 'default';
  children: React.ReactNode;
}) {
  const badgeClassName = cn(
    'px-2 py-0.5 flex items-center rounded-sm font-medium bg-slate-200 text-sm text-slate-600',
    variant === 'primary' && 'bg-indigo-500 text-white',
    className
  );
  return <span className={badgeClassName}>{children}</span>;
}
