export type App = {
  id: string;
  name: string;
  sdkVersion: string;
  framework: string | null;
  language?: string;
  syncMethod: SyncMethod;
  lastSyncedAt?: Date;
  url: string | null;
};

export const syncMethods = ['PERSISTENT', 'SERVERLESS'] as const;

export type SyncMethod = (typeof syncMethods)[number];

export type AppKind = 'default' | 'info' | 'warning' | 'primary' | 'error';
