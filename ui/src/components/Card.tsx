import { type PropsWithChildren } from 'react';

import classNames from '@/utils/classnames';

export function Card({
  accentColor,
  children,
  className,
}: PropsWithChildren<{ accentColor?: string; className?: string }>) {
  return (
    <div className={classNames('w-full bg-slate-950 rounded-lg shadow overflow-hidden', className)}>
      {accentColor && <div className={classNames('pt-2', accentColor)} />}
      {children}
    </div>
  );
}

Card.Content = ({ children }: PropsWithChildren) => {
  return <div className="bg-slate-800/40 p-2">{children}</div>;
};

Card.Header = ({ children }: PropsWithChildren) => {
  return (
    <div className="bg-slate-800/40 flex flex-col gap-1 px-3 py-3 border-b border-slate-700 text-white">
      {children}
    </div>
  );
};
