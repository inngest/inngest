import {
  transformFramework,
  transformLanguage,
  transformPlatform,
} from '@inngest/components/utils/appsParser';

import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query App($envID: ID!, $externalAppID: String!) {
    environment: workspace(id: $envID) {
      app: appByExternalID(externalID: $externalAppID) {
        id
        externalID
        functions {
          id
          latestSyncedConfig {
            triggers {
              type
              value
            }
          }
          name
          slug
        }
        appVersion
        name
        method
        latestSync {
          commitAuthor
          commitHash
          commitMessage
          commitRef
          error
          framework
          id
          lastSyncedAt
          platform
          repoURL
          sdkLanguage
          sdkVersion
          status
          url
          vercelDeploymentID
          vercelDeploymentURL
          vercelProjectID
          vercelProjectURL
          appVersion
        }
      }
    }
  }
`);

export function useApp({ envID, externalAppID }: { envID: string; externalAppID: string }) {
  const res = useGraphQLQuery({
    pollIntervalInMilliseconds: 10_000,
    query,
    variables: { envID, externalAppID },
  });

  if (res.data) {
    const { app } = res.data.environment;
    let latestSync = null;
    if (app.latestSync) {
      latestSync = {
        ...app.latestSync,
        lastSyncedAt: new Date(app.latestSync.lastSyncedAt),
        framework: transformFramework(app.latestSync.framework),
        platform: transformPlatform(app.latestSync.platform),
        sdkLanguage: transformLanguage(app.latestSync.sdkLanguage),
      };
    }

    return {
      ...res,
      data: {
        ...app,
        functions: app.functions.map((fn) => {
          return {
            ...fn,
            triggers: fn.latestSyncedConfig?.triggers || [],
          };
        }),
        latestSync,
      },
    };
  }

  return { ...res, data: undefined };
}
