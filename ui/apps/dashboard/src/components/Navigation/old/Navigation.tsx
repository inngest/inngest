import cn from '@/utils/cn';

export default function Navigation({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <nav className={cn('flex items-center gap-1 md:gap-3 lg:gap-1 xl:gap-3', className)}>
      {children}
    </nav>
  );
}
