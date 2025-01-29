export const workerStatuses = [
  'CONNECTED',
  'DISCONNECTED',
  'DISCONNECTING',
  'DRAINING',
  'READY',
] as const;
type WorkerStatus = (typeof workerStatuses)[number];

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

export type Worker = {
  appVersion: string;
  connectedAt: Date;
  cpuCores: number;
  id: string;
  instanceID: string | null;
  lastHeartbeatAt: Date | null;
  memBytes: number;
  os: string;
  sdkLang: string;
  sdkVersion: string;
  workerIp: string;
  status: GroupedWorkerStatus;
  functionCount: number;
};

export const groupedWorkerStatuses = ['ACTIVE', 'INACTIVE', 'DISCONNECTED'] as const;

export type GroupedWorkerStatus = (typeof groupedWorkerStatuses)[number];

export function isWorkerStatus(s: string): s is GroupedWorkerStatus {
  return groupedWorkerStatuses.includes(s as GroupedWorkerStatus);
}
