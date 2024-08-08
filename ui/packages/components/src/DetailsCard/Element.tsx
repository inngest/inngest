import { InlineCode } from '@inngest/components/InlineCode';
import { Link, type LinkProps } from '@inngest/components/Link';
import { Skeleton } from '@inngest/components/Skeleton';
import { Time } from '@inngest/components/Time';
import { cn } from '@inngest/components/utils/classNames';

import { isLazyDone, type Lazy } from '../utils/lazyLoad';

const cellStyles = 'text-basis text-sm';

export function ElementWrapper({
  label,
  children,
  className,
}: React.PropsWithChildren<{ label: string; className?: string }>) {
  return (
    <div className={cn('w-64 text-sm', className)}>
      <dt className="text-subtle text-xs">{label}</dt>
      <dd className="truncate">{children}</dd>
    </div>
  );
}

export function LazyElementWrapper<T>({
  children,
  className,
  label,
  lazy,
}: {
  children: (loaded: T) => React.ReactNode;
  className?: string;
  label: string;
  lazy: Lazy<T>;
}) {
  let content;
  if (isLazyDone(lazy)) {
    content = children(lazy);
  } else {
    content = <SkeletonElement />;
  }

  return (
    <ElementWrapper label={label} className={className}>
      {content}
    </ElementWrapper>
  );
}

// Optimistically render an initial value while waiting for a
// lazy loaded value to render the final component
export function OptimisticElementWrapper<T, InitialType>({
  children,
  optimisticChildren,
  className,
  label,
  lazy,
  initial,
}: {
  children: (loaded: T) => React.ReactNode;
  optimisticChildren: (loaded: InitialType) => React.ReactNode;
  className?: string;
  label: string;
  lazy: Lazy<T>;
  initial?: InitialType;
}) {
  let content;
  if (isLazyDone(lazy)) {
    content = children(lazy);
  } else if (initial) {
    content = optimisticChildren(initial);
  } else {
    content = <SkeletonElement />;
  }

  return (
    <ElementWrapper label={label} className={className}>
      {content}
    </ElementWrapper>
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
