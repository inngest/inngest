import { type PropsWithChildren } from 'react';
import { classNames } from '@inngest/components/utils/classNames';

export function Card({
  accentColor,
  children,
  className,
}: PropsWithChildren<{ accentColor?: string; className?: string }>) {
  return (
    <div
      className={classNames(
        'bg-slate-910 w-full overflow-hidden rounded-lg border border-slate-700/30 shadow',
        className
      )}
    >
      {accentColor && <div className={classNames('pt-2', accentColor)} />}
      {children}
    </div>
  );
}

Card.Content = ({ children }: PropsWithChildren) => {
  return <div className="bg-slate-800/40 p-2.5">{children}</div>;
};

Card.Header = ({ children }: PropsWithChildren) => {
  return (
    <div className="flex flex-col gap-1 border-b border-slate-800/50 bg-slate-800/40 px-4 py-2.5 text-sm text-slate-400">
      {children}
    </div>
  );
};
