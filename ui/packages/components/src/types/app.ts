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
  appVersion?: string | null;
};

export const methodTypes = {
  Api: 'API',
  Connect: 'CONNECT',
  Serve: 'SERVE',
} as const;

export type ConnectionType = (typeof methodTypes)[keyof typeof methodTypes];

export type AppKind = 'default' | 'info' | 'warning' | 'primary' | 'error';

export const AppStatuses = ['ACTIVE', 'ARCHIVED'] as const;

export type AppStatus = (typeof AppStatuses)[number];

export function isAppStatus(s: string): s is AppStatus {
  return AppStatuses.includes(s as AppStatus);
}
