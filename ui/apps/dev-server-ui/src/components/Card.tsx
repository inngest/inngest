import { type PropsWithChildren } from 'react';

import classNames from '@/utils/classnames';

export function Card({
  accentColor,
  children,
  className,
}: PropsWithChildren<{ accentColor?: string; className?: string }>) {
  return (
    <div
      className={classNames(
        'w-full bg-slate-910 rounded-lg shadow overflow-hidden border border-slate-700/30',
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
    <div className="bg-slate-800/40 flex flex-col gap-1 px-4 py-2.5 border-b border-slate-800/50 text-slate-400 text-sm">
      {children}
    </div>
  );
};
