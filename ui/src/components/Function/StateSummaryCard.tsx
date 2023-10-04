import { type PropsWithChildren } from 'react';

import classNames from '@/utils/classnames';

export function StateSummaryCard({
  children,
  className,
}: PropsWithChildren<{ className?: string }>) {
  return (
    <div className={classNames('w-full bg-slate-950 rounded-lg shadow overflow-hidden', className)}>
      {children}
    </div>
  );
}

StateSummaryCard.Accent = function Content({ className }: { className: string }) {
  return <div className={classNames('pt-3', className)} />;
};

StateSummaryCard.Content = function Content({ children }: PropsWithChildren) {
  return <div className="p-2.5">{children}</div>;
};

StateSummaryCard.Header = function Content({ children }: PropsWithChildren) {
  return (
    <div className="flex flex-col gap-1 px-5 py-3 border-b border-slate-700/30 text-white">
      {children}
    </div>
  );
};
