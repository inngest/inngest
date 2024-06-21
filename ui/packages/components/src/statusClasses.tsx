import {
  isFunctionRunStatus,
  isReplayRunStatus,
  type FunctionRunStatus,
  type ReplayRunStatus,
} from './types/functionRun';
import { cn } from './utils/classNames';

const backgroundClasses: { [key in FunctionRunStatus | ReplayRunStatus | 'UNKNOWN']: string } = {
  CANCELLED: 'bg-status-cancelled',
  COMPLETED: 'bg-status-completed',
  FAILED: 'bg-status-failed',
  QUEUED: 'bg-status-queued',
  RUNNING: 'bg-status-running',
  UNKNOWN: 'bg-status-cancelled',
  SKIPPED_PAUSED: 'bg-accent-moderate',
};

export function getStatusBackgroundClass(status: string): string {
  if (!isFunctionRunStatus(status) && !isReplayRunStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return backgroundClasses['UNKNOWN'];
  }
  return backgroundClasses[status];
}

const borderClasses: { [key in FunctionRunStatus | ReplayRunStatus | 'UNKNOWN']: string } = {
  CANCELLED: 'border-status-cancelled',
  COMPLETED: 'border-status-completed',
  FAILED: 'border-status-failed',
  QUEUED: 'border-status-queued',
  RUNNING: 'border-status-running',
  UNKNOWN: 'border-status-cancelled',
  SKIPPED_PAUSED: 'border-accent-moderate',
};

export function getStatusBorderClass(status: string): string {
  if (!isFunctionRunStatus(status) && !isReplayRunStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return cn('border', borderClasses['UNKNOWN']);
  }
  return cn('border', borderClasses[status]);
}

const textClasses: { [key in FunctionRunStatus | ReplayRunStatus | 'UNKNOWN']: string } = {
  CANCELLED: 'text-status-cancelled',
  COMPLETED: 'text-status-completed',
  FAILED: 'text-status-failed',
  QUEUED: 'text-status-queued',
  RUNNING: 'text-status-running',
  UNKNOWN: 'text-status-cancelled',
  SKIPPED_PAUSED: 'text-accent-moderate',
};

export function getStatusTextClass(status: string): string {
  if (!isFunctionRunStatus(status) && !isReplayRunStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return textClasses['UNKNOWN'];
  }
  return textClasses[status];
}
