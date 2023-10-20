import cn from '@/utils/cn';

export default function Navigation({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <nav className={cn('flex items-center gap-3', className)}>{children}</nav>;
}
