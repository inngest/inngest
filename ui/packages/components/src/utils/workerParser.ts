import { type GroupedWorkerStatus, type WorkerStatus } from '@inngest/components/types/workers';

// Util to convert filter status into API status
export const convertGroupedWorkerStatusToWorkerStatuses = (
  groupedStatus: GroupedWorkerStatus
): WorkerStatus[] => {
  switch (groupedStatus) {
    case 'ACTIVE':
      return ['READY'];
    case 'DISCONNECTED':
      return ['DISCONNECTED'];
    case 'INACTIVE':
      return ['DISCONNECTING', 'CONNECTED', 'DRAINING'];
    default:
      return [groupedStatus];
  }
};

// We only display three statuses for workers: ACTIVE, INACTIVE, and DISCONNECTED
export const convertWorkerStatus = (status: WorkerStatus): GroupedWorkerStatus => {
  switch (status) {
    case 'READY':
      return 'ACTIVE';
    case 'DISCONNECTED':
      return 'DISCONNECTED';
    case 'DISCONNECTING':
    case 'CONNECTED':
    case 'DRAINING':
      return 'INACTIVE';
    default:
      return status;
  }
};
