export type App = {
  id: string;
  name: string;
  sdkVersion?: string;
  framework?: string | null;
  sdkLanguage?: string;
  connectionType?: ConnectionType;
  lastSyncedAt?: Date;
  url?: string | null;
};

export const connectionTypes = {
  Connect: 'CONNECT',
  Serverless: 'SERVERLESS',
} as const;

export type ConnectionType = (typeof connectionTypes)[keyof typeof connectionTypes];

export type AppKind = 'default' | 'info' | 'warning' | 'primary' | 'error';
