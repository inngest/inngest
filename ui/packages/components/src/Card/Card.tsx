import { type PropsWithChildren } from 'react';
import { cn } from '@inngest/components/utils/classNames';

type Props = PropsWithChildren<{
  accentColor?: string;
  accentPosition?: 'left' | 'top';
  className?: string;
}>;

export function Card({ accentColor, accentPosition = 'top', children, className }: Props) {
  // Need some dynamic classes to properly handle the accent's existence and
  // position
  let accentClass = undefined;
  let contentClass = 'rounded-md border';
  let wrapperClass = undefined;

  if (accentColor) {
    if (accentPosition === 'left') {
      // The left border is the responsibility of the accent
      accentClass = 'rounded-l-md';
      contentClass = 'rounded-r-md border-l-0';

      // Need flex to move the accent to the left
      wrapperClass = 'flex';
    } else if (accentPosition === 'top') {
      // The top border is the responsibility of the accent
      accentClass = 'rounded-t-md';
      contentClass = 'rounded-b-md border-t-0';
    }
  }

  return (
    <div
      className={cn(
        'dark:bg-slate-910 w-full overflow-hidden dark:shadow',
        wrapperClass,
        className
      )}
    >
      {accentColor && <div className={cn('p-0.5', accentClass, accentColor)} />}
      <div className={cn('w-full grow border border-slate-300', contentClass)}>{children}</div>
    </div>
  );
}

Card.Content = ({ children, className }: PropsWithChildren<{ className?: string }>) => {
  return <div className={cn('p-2.5 dark:bg-slate-800/40', className)}>{children}</div>;
};

Card.Footer = ({ children, className }: PropsWithChildren<{ className?: string }>) => {
  return (
    <div
      className={cn(
        'border-t border-slate-300 px-4 py-2 dark:border-slate-800/50 dark:bg-slate-800/40',
        className
      )}
    >
      {children}
    </div>
  );
};

Card.Header = ({ children, className }: PropsWithChildren<{ className?: string }>) => {
  return (
    <div
      className={cn(
        'flex flex-col gap-1 border-b border-slate-300 px-4 py-2.5 text-sm text-slate-700 dark:border-slate-800/50 dark:bg-slate-800/40 dark:text-slate-400',
        className
      )}
    >
      {children}
    </div>
  );
};
