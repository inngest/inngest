import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query Apps($envID: ID!) {
    environment: workspace(id: $envID) {
      apps {
        id
        externalID
        name
        latestSync {
          createdAt
          framework
          id
          platform
          sdkLanguage
          sdkVersion
          status
          syncedFunctions: deployedFunctions {
            id
          }
          url
        }
      }
    }
  }
`);

export function useApps(envID: string) {
  const res = useGraphQLQuery({
    pollIntervalInMilliseconds: 10_000,
    query,
    variables: { envID },
  });

  if (res.data) {
    const apps = res.data.environment.apps.map((app) => {
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
      };
    });

    return {
      ...res,
      data: apps,
    };
  }

  return res;
}
