import { RunStatusIcon } from '@inngest/components/FunctionRunStatusIcon/RunStatusIcons';
import { type FunctionRunStatus } from '@inngest/components/types/functionRun';
import { cn } from '@inngest/components/utils/classNames';

const cellStyles = 'text-slate-950 text-sm';

export function IDCell({ children }: React.PropsWithChildren) {
  return <p className={cn(cellStyles, 'font-mono')}>{children}</p>;
}

export function TextCell({ children }: React.PropsWithChildren) {
  return <p className={cn(cellStyles, 'font-medium')}>{children}</p>;
}

export function TimeCell({ children }: React.PropsWithChildren) {
  // TODO: Move Time component from Cloud to shared components, to use here
  return <span className={cn(cellStyles, 'font-medium')}>{children}</span>;
}

export function StatusCell({ status }: React.PropsWithChildren<{ status: FunctionRunStatus }>) {
  const statusStyles: Record<string, string> = {
    CANCELLED: 'text-neutral-600',
    COMPLETED: 'text-teal-700',
    FAILED: 'text-rose-500',
    RUNNING: 'text-sky-500',
    QUEUED: 'text-amber-500',
  } as const satisfies { [key in FunctionRunStatus]: string };
  const style = statusStyles[status] ?? statusStyles['CANCELLED'];

  return (
    <div className={cn(cellStyles, 'flex items-center gap-2.5 font-medium')}>
      <RunStatusIcon status={status} />
      <p className={cn(style, 'lowercase first-letter:capitalize')}>{status}</p>
    </div>
  );
}
