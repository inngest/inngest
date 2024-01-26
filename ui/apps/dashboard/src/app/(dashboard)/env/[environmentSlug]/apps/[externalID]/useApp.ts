import type { Function } from '@inngest/components/types/function';

import { graphql } from '@/gql';
import { useGraphQLQuery_TEMPORARY } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query App($envID: ID!, $externalAppID: String!) {
    environment: workspace(id: $envID) {
      app: appByExternalID(externalID: $externalAppID) {
        id
        externalID
        functions {
          id
          latestVersion {
            triggers {
              eventName
              schedule
            }
          }
          name
          slug
        }
        name
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
        }
      }
    }
  }
`);

export function useApp({ envID, externalAppID }: { envID: string; externalAppID: string }) {
  const res = useGraphQLQuery_TEMPORARY({
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
      };
    }

    return {
      ...res,
      data: {
        ...app,
        functions: app.functions.map((fn) => {
          return {
            ...fn,
            triggers: transformTriggers(fn.latestVersion.triggers),
          };
        }),
        latestSync,
      },
    };
  }

  return { ...res, data: undefined };
}

function transformTriggers(
  rawTriggers: { eventName: string | null; schedule: string | null }[]
): Function['triggers'] {
  const triggers: Function['triggers'] = [];

  for (const trigger of rawTriggers) {
    if (trigger.eventName) {
      triggers.push({
        type: 'EVENT',
        value: trigger.eventName,
      });
    } else if (trigger.schedule) {
      triggers.push({
        type: 'CRON',
        value: trigger.schedule,
      });
    }
  }

  return triggers;
}
