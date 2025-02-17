export type App = {
  id: string;
  name: string;
  sdkVersion?: string;
  framework?: string | null;
  sdkLanguage?: string;
  method?: ConnectionType;
  lastSyncedAt?: Date;
  url?: string | null;
  externalID?: string;
  version?: string;
};

export const methodTypes = {
  Connect: 'CONNECT',
  Serve: 'SERVE',
} as const;

export type ConnectionType = (typeof methodTypes)[keyof typeof methodTypes];

export type AppKind = 'default' | 'info' | 'warning' | 'primary' | 'error';
