import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query Apps($envID: ID!) {
    environment: workspace(id: $envID) {
      apps {
        id
        externalID
        functionCount
        name
        latestSync {
          createdAt
          framework
          id
          platform
          sdkLanguage
          sdkVersion
          status
          url
        }
      }

      unattachedSyncs(first: 1) {
        createdAt
      }
    }
  }
`);

export function useApps({ envID, isArchived }: { envID: string; isArchived: boolean }) {
  const res = useGraphQLQuery({
    pollIntervalInMilliseconds: 10_000,
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
            createdAt: new Date(app.latestSync.createdAt),
          };
        }

        return {
          ...app,
          latestSync,

          // This is a hack to get around the fact that app archival is not a
          // first-class feature yet. We'll infer that an app is archived if all
          // of its functions are archived.
          isArchived: app.functionCount === 0,
        };
      })
      .filter((app) => {
        // Filter the results because GraphQL doesn't have an isArchived filter
        // yet.
        return app.latestSync && app.isArchived === isArchived;
      });

    let latestUnattachedSyncTime;
    if (res.data.environment.unattachedSyncs[0]) {
      latestUnattachedSyncTime = new Date(res.data.environment.unattachedSyncs[0].createdAt);
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
