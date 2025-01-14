export const workerStatuses = [
  'CONNECTED',
  'DISCONNECTED',
  'DISCONNECTING',
  'DRAINING',
  'READY',
] as const;
type WorkerStatus = (typeof workerStatuses)[number];

// We only display three statuses for workers: ACTIVE, INACTIVE, and FAILED
export const convertWorkerStatus = (status: WorkerStatus): GroupedWorkerStatus | 'UNKNOWN' => {
  switch (status) {
    case 'READY':
      return 'ACTIVE';
    case 'DISCONNECTED':
      return 'FAILED';
    case 'DISCONNECTING':
    case 'CONNECTED':
    case 'DRAINING':
      return 'INACTIVE';
    default:
      return 'UNKNOWN';
  }
};

export type Worker = {
  appVersion: string;
  appID: string;
  connectedAt: Date;
  cpuCores: number;
  disconnectedAt: Date | null;
  gatewayID: string;
  groupHash: string;
  id: string;
  instanceID: string | null;
  lastHeartbeatAt: Date | null;
  memBytes: number;
  os: string;
  sdkLang: string;
  sdkPlatform: string;
  sdkVersion: string;
  status: GroupedWorkerStatus;
  syncID: string | null;
};

export const groupedWorkerStatuses = ['ACTIVE', 'INACTIVE', 'FAILED'] as const;

export type GroupedWorkerStatus = (typeof groupedWorkerStatuses)[number];

export function isWorkerStatus(s: string): s is GroupedWorkerStatus {
  return groupedWorkerStatuses.includes(s as GroupedWorkerStatus);
}
