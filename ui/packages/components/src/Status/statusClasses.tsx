import {
  isFunctionRunStatus,
  isReplayRunStatus,
  type FunctionRunStatus,
  type ReplayRunStatus,
} from '../types/functionRun';
import { isWorkerStatus, type GroupedWorkerStatus } from '../types/workers';
import { cn } from '../utils/classNames';

const backgroundClasses: {
  [key in FunctionRunStatus | ReplayRunStatus | GroupedWorkerStatus | 'UNKNOWN']: string;
} = {
  CANCELLED: 'bg-status-cancelled',
  COMPLETED: 'bg-status-completed',
  FAILED: 'bg-status-failed',
  QUEUED: 'bg-status-queuedSubtle',
  RUNNING: 'bg-status-runningSubtle',
  UNKNOWN: 'bg-status-cancelled',
  SKIPPED_PAUSED: 'bg-accent-intense',
  PAUSED: 'bg-status-paused',
  SKIPPED: 'bg-status-paused',
  INACTIVE: 'bg-accent-subtle dark:bg-accent-intense',
  ACTIVE: 'bg-status-completed',
  DISCONNECTED: 'bg-status-cancelled',
};

export function getStatusBackgroundClass(status: string): string {
  if (!isFunctionRunStatus(status) && !isReplayRunStatus(status) && !isWorkerStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return backgroundClasses['UNKNOWN'];
  }
  return backgroundClasses[status];
}

const borderClasses: {
  [key in FunctionRunStatus | ReplayRunStatus | GroupedWorkerStatus | 'UNKNOWN']: string;
} = {
  CANCELLED: 'border-status-cancelled',
  COMPLETED: 'border-status-completed',
  FAILED: 'border-status-failed',
  QUEUED: 'border-status-queued',
  RUNNING: 'border-status-running',
  UNKNOWN: 'border-status-cancelled',
  SKIPPED_PAUSED: 'border-accent-intense',
  PAUSED: 'border-accent-paused',
  SKIPPED: 'border-accent-paused',
  INACTIVE: 'border-accent-subtle dark:border-accent-intense',
  ACTIVE: 'border-status-completed',
  DISCONNECTED: 'border-status-cancelled',
};

export function getStatusBorderClass(status: string): string {
  if (!isFunctionRunStatus(status) && !isReplayRunStatus(status) && !isWorkerStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return cn('border', borderClasses['UNKNOWN']);
  }
  return cn('border', borderClasses[status]);
}

const textClasses: {
  [key in FunctionRunStatus | ReplayRunStatus | GroupedWorkerStatus | 'UNKNOWN']: string;
} = {
  CANCELLED: 'text-status-cancelledText',
  COMPLETED: 'text-status-completedText',
  FAILED: 'text-status-failedText',
  QUEUED: 'text-status-queuedText',
  RUNNING: 'text-status-runningText',
  UNKNOWN: 'text-status-cancelledText',
  SKIPPED_PAUSED: 'text-accent-intense',
  PAUSED: 'text-status-pausedText',
  SKIPPED: 'text-status-pausedText',
  INACTIVE: 'text-accent-subtle dark:text-accent-intense',
  ACTIVE: 'text-status-completedText',
  DISCONNECTED: 'text-status-cancelledText',
};

export function getStatusTextClass(status: string): string {
  if (!isFunctionRunStatus(status) && !isReplayRunStatus(status) && !isWorkerStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return textClasses['UNKNOWN'];
  }
  return textClasses[status];
}
