import cn from '@/utils/cn';

export default function ListContainer({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <ul
      className={cn(
        'divide-y divide-solid divide-slate-200 overflow-hidden rounded-lg border border-slate-200',
        className
      )}
    >
      {children}
    </ul>
  );
}
