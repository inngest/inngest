import {
  isFunctionRunStatus,
  isReplayRunStatus,
  type FunctionRunStatus,
  type ReplayRunStatus,
} from '../types/functionRun';
import { isWorkerStatus, type WorkerStatus } from '../types/workers';
import { cn } from '../utils/classNames';

const backgroundClasses: {
  [key in FunctionRunStatus | ReplayRunStatus | WorkerStatus | 'UNKNOWN']: string;
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
  CONNECTED: 'bg-accent-intense',
  DISCONNECTED: 'bg-status-failed',
  DISCONNECTING: 'bg-accent-intense',
  DRAINING: 'bg-accent-intense',
  READY: 'bg-status-completed',
};

export function getStatusBackgroundClass(status: string): string {
  if (!isFunctionRunStatus(status) && !isReplayRunStatus(status) && !isWorkerStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return backgroundClasses['UNKNOWN'];
  }
  return backgroundClasses[status];
}

const borderClasses: {
  [key in FunctionRunStatus | ReplayRunStatus | WorkerStatus | 'UNKNOWN']: string;
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
  CONNECTED: 'border-accent-intense',
  DISCONNECTED: 'border-status-failed',
  DISCONNECTING: 'border-accent-intense',
  DRAINING: 'border-accent-intense',
  READY: 'border-status-completed',
};

export function getStatusBorderClass(status: string): string {
  if (!isFunctionRunStatus(status) && !isReplayRunStatus(status) && !isWorkerStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return cn('border', borderClasses['UNKNOWN']);
  }
  return cn('border', borderClasses[status]);
}

const textClasses: {
  [key in FunctionRunStatus | ReplayRunStatus | WorkerStatus | 'UNKNOWN']: string;
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
  CONNECTED: 'text-accent-intense',
  DISCONNECTED: 'text-status-failedText',
  DISCONNECTING: 'text-accent-intense',
  DRAINING: 'text-accent-intense',
  READY: 'text-status-completedText',
};

export function getStatusTextClass(status: string): string {
  if (!isFunctionRunStatus(status) && !isReplayRunStatus(status) && !isWorkerStatus(status)) {
    console.error(`unexpected status: ${status}`);
    return textClasses['UNKNOWN'];
  }
  return textClasses[status];
}
