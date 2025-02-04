import type { Function } from '@inngest/components/types/function';

import { graphql } from '@/gql';
import { type AppsQuery } from '@/gql/graphql';
import { transformTriggers } from '@/utils/triggers';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

export type FlattenedApp = Omit<
  AppsQuery['environment']['apps'][number],
  'latestSync' | 'functions'
> & {
  __typename?: 'App';
  lastSyncedAt?: Date;
  error?: string | null;
  framework?: string | null;
  platform?: string | null;
  sdkLanguage?: string;
  sdkVersion?: string;
  status?: string;
  url?: string | null;
  functions: Function[];
};

const query = graphql(`
  query Apps($envID: ID!) {
    environment: workspace(id: $envID) {
      apps {
        id
        externalID
        functionCount
        isArchived
        name
        connectionType
        isParentArchived
        latestSync {
          error
          framework
          id
          lastSyncedAt
          platform
          sdkLanguage
          sdkVersion
          status
          url
        }
        functions {
          id
          name
          slug
          triggers {
            eventName
            schedule
          }
        }
      }

      unattachedSyncs(first: 1) {
        lastSyncedAt
      }
    }
  }
`);

export function useApps({ envID, isArchived }: { envID: string; isArchived: boolean }) {
  const res = useGraphQLQuery({
    pollIntervalInMilliseconds: 2_000,
    query,
    variables: { envID },
  });

  // We are flattening the latestSync data to match the structure used in the DevServer
  if (res.data) {
    const apps = res.data.environment.apps
      .map(({ latestSync, functions, ...app }) => {
        const latestSyncData: Omit<FlattenedApp, keyof typeof app | 'functions'> = latestSync
          ? {
              lastSyncedAt: new Date(latestSync.lastSyncedAt),
              error: latestSync.error,
              framework: latestSync.framework,
              platform: latestSync.platform,
              sdkLanguage: latestSync.sdkLanguage,
              sdkVersion: latestSync.sdkVersion,
              status: latestSync.status,
              url: latestSync.url,
            }
          : {
              lastSyncedAt: undefined,
              error: undefined,
              framework: undefined,
              platform: undefined,
              sdkLanguage: undefined,
              sdkVersion: undefined,
              status: undefined,
              url: undefined,
            };

        return {
          ...app,
          ...latestSyncData,
          functions: functions.map((fn) => {
            return {
              ...fn,
              triggers: transformTriggers(fn.triggers),
            };
          }),
          __typename: 'App' as const,
        };
      })
      .filter((app) => app.lastSyncedAt && app.isArchived === isArchived);

    let latestUnattachedSyncTime;
    if (res.data.environment.unattachedSyncs[0]) {
      latestUnattachedSyncTime = new Date(res.data.environment.unattachedSyncs[0].lastSyncedAt);
    }

    return {
      ...res,
      data: {
        apps,
        latestUnattachedSyncTime,
      },
    };
  }

  return {
    ...res,
    data: undefined,
  };
}
