import { isFunctionRunStatus, type FunctionRunStatus } from './types/functionRun';
import { cn } from './utils/classNames';

const backgroundClasses: { [key in FunctionRunStatus | 'UNKNOWN']: string } = {
  CANCELLED: 'bg-slate-300',
  COMPLETED: 'bg-teal-500',
  FAILED: 'bg-rose-500',
  QUEUED: 'bg-amber-100',
  RUNNING: 'bg-blue-200',
  UNKNOWN: 'bg-slate-300',
};

export function getStatusBackgroundClass(status: string): string {
  if (!isFunctionRunStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return backgroundClasses['UNKNOWN'];
  }
  return backgroundClasses[status];
}

const borderClasses: { [key in FunctionRunStatus | 'UNKNOWN']: string } = {
  CANCELLED: 'border-slate-300',
  COMPLETED: 'border-teal-500',
  FAILED: 'border-rose-500',
  QUEUED: 'border-amber-500',
  RUNNING: 'border-sky-500',
  UNKNOWN: 'border-slate-300',
};

export function getStatusBorderClass(status: string): string {
  if (!isFunctionRunStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return cn('border', borderClasses['UNKNOWN']);
  }
  return cn('border', borderClasses[status]);
}

const textClasses: { [key in FunctionRunStatus | 'UNKNOWN']: string } = {
  CANCELLED: 'text-neutral-600',
  COMPLETED: 'text-teal-700',
  FAILED: 'text-rose-500',
  RUNNING: 'text-sky-500',
  QUEUED: 'text-amber-500',
  UNKNOWN: 'text-neutral-600',
};

export function getStatusTextClass(status: string): string {
  if (!isFunctionRunStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return textClasses['UNKNOWN'];
  }
  return textClasses[status];
}
