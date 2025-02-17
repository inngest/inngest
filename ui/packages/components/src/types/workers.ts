export enum workerStatuses {
  Connected = 'CONNECTED',
  Disconnected = 'DISCONNECTED',
  Disconnecting = 'DISCONNECTING',
  Draining = 'DRAINING',
  Ready = 'READY',
}

export type WorkerStatus = `${workerStatuses}`;

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
  connectedAt: string;
  cpuCores: number;
  id: string;
  instanceID: string | null;
  lastHeartbeatAt: string | null;
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

export type ConnectV1WorkerConnectionsOrderBy = {
  direction: ConnectV1WorkerConnectionsOrderByDirection;
  field: ConnectV1WorkerConnectionsOrderByField;
};

export enum ConnectV1WorkerConnectionsOrderByDirection {
  Asc = 'ASC',
  Desc = 'DESC',
}

export enum ConnectV1WorkerConnectionsOrderByField {
  ConnectedAt = 'CONNECTED_AT',
  DisconnectedAt = 'DISCONNECTED_AT',
  LastHeartbeatAt = 'LAST_HEARTBEAT_AT',
}

export type PageInfo = {
  endCursor: string | null;
  hasNextPage: boolean;
  hasPreviousPage: boolean;
  startCursor: string | null;
};
