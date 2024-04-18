import { cn } from '../utils/classNames';

const cellStyles = 'text-slate-950 text-sm';

export function IDCell({ children }: { children: React.ReactNode }) {
  return <p className={cn(cellStyles, 'font-mono')}>{children}</p>;
}

export function TextCell({ children }: { children: React.ReactNode }) {
  return <p className={cn(cellStyles, 'font-medium')}>{children}</p>;
}

export function TimeCell({ children }: { children: React.ReactNode }) {
  // TODO: Move Time component from Cloud to shared components, to use here
  return <span className={cn(cellStyles, 'font-medium')}>{children}</span>;
}

export function StatusCell({ status, children }: { status: string; children: React.ReactNode }) {
  // TODO: Use new runs circles and colors instead of passing FunctionRunStatusIcon as children
  return (
    <p className={cn(cellStyles, 'flex items-center gap-2.5 font-medium')}>
      {children}
      <p className="lowercase first-letter:capitalize">{status}</p>
    </p>
  );
}
