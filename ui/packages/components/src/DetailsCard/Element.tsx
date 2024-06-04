import { InlineCode } from '@inngest/components/InlineCode';
import { Link, type LinkProps } from '@inngest/components/Link';
import { Skeleton } from '@inngest/components/Skeleton';
import { Time } from '@inngest/components/Time';
import { cn } from '@inngest/components/utils/classNames';

const cellStyles = 'text-slate-700 text-sm';

export function ElementWrapper({
  label,
  children,
  className,
}: React.PropsWithChildren<{ label: string; className?: string }>) {
  return (
    <div className={cn('w-64 text-sm', className)}>
      <dt className="pb-2 text-slate-500">{label}</dt>
      <dd className="truncate">{children}</dd>
    </div>
  );
}

export function IDElement({ children }: React.PropsWithChildren) {
  return <span className={cn(cellStyles, 'font-mono')}>{children}</span>;
}

export function TextElement({ children }: React.PropsWithChildren) {
  return <span className={cn(cellStyles, 'font-medium')}>{children}</span>;
}

export function TimeElement({ date }: { date: Date }) {
  return (
    <span className={cn(cellStyles, 'font-medium')}>
      <Time value={date} />
    </span>
  );
}

export function LinkElement({ children, href, ...props }: LinkProps) {
  return (
    <Link href={href} {...props}>
      {children}
    </Link>
  );
}

export function CodeElement({ value }: { value: string }) {
  return <InlineCode value={value} />;
}

export function SkeletonElement() {
  return <Skeleton className="h-5 w-full" />;
}
