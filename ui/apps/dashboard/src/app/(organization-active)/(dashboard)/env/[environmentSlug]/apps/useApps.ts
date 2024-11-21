import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query Apps($envID: ID!) {
    environment: workspace(id: $envID) {
      apps {
        id
        externalID
        functionCount
        isArchived
        name
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

  if (res.data) {
    const apps = res.data.environment.apps
      .map((app) => {
        let latestSync = null;
        if (app.latestSync) {
          latestSync = {
            ...app.latestSync,
            lastSyncedAt: new Date(app.latestSync.lastSyncedAt),
          };
        }

        return {
          ...app,
          latestSync,
          isArchived: app.isArchived,
        };
      })
      .filter((app) => {
        // Filter the results because GraphQL doesn't have an isArchived filter
        // yet.
        return app.latestSync && app.isArchived === isArchived;
      });

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
