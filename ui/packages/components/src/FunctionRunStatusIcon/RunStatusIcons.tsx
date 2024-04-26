import { type FunctionRunStatus } from '@inngest/components/types/functionRun';

import { cn } from '../utils/classNames';

const statusStyles: Record<string, string> = {
  CANCELLED: 'bg-slate-300 border-slate-300',
  COMPLETED: 'bg-teal-500 border-teal-500',
  FAILED: 'bg-rose-500 border-rose-500',
  RUNNING: 'bg-blue-200 border-sky-500',
  QUEUED: 'bg-amber-100 border-amber-500',
} as const satisfies { [key in FunctionRunStatus]: string };

type Props = {
  status: FunctionRunStatus;
  className?: string;
};

export function RunStatusIcon({ status, className }: Props) {
  const style = statusStyles[status] ?? statusStyles['CANCELLED'];

  const title = 'Function ' + status.toLowerCase();
  return <div className={cn(className, style, 'h-3.5 w-3.5 rounded-full border')} title={title} />;
}
