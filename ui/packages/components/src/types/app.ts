export type App = {
  id: string;
  name: string;
  sdkVersion: string;
  framework: string | null;
  sdkLanguage?: string;
  connectionType?: ConnectionType;
  lastSyncedAt?: Date;
  url: string | null;
};

export const connectionTypes = ['CONNECT', 'SERVERLESS'] as const;

export type ConnectionType = (typeof connectionTypes)[number];

export type AppKind = 'default' | 'info' | 'warning' | 'primary' | 'error';
