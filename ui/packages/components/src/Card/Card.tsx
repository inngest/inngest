import { type PropsWithChildren } from 'react';

import { cn } from '../utils/classNames';

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
    <div className={cn('bg-canvasBase w-full overflow-hidden rounded-md', wrapperClass, className)}>
      {accentColor && <div className={cn('p-0.5', accentClass, accentColor)} />}
      <div className={cn('border-subtle w-full grow overflow-hidden border', contentClass)}>
        {children}
      </div>
    </div>
  );
}

Card.Content = ({ children, className }: PropsWithChildren<{ className?: string }>) => {
  return <div className={cn('bg-canvasBase px-6 py-4 ', className)}>{children}</div>;
};

Card.Footer = ({ children, className }: PropsWithChildren<{ className?: string }>) => {
  return (
    <div className={cn('border-subtle bg-canvasBase border-t px-6 py-3', className)}>
      {children}
    </div>
  );
};

Card.Header = ({ children, className }: PropsWithChildren<{ className?: string }>) => {
  return (
    <div
      className={cn(
        'border-subtle bg-canvasBase text-basis flex flex-col gap-1 border-b py-3 pl-6 pr-4 text-sm',
        className
      )}
    >
      {children}
    </div>
  );
};
