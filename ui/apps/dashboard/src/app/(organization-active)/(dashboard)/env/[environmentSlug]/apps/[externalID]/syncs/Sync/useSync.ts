import {
  transformFramework,
  transformLanguage,
  transformPlatform,
} from '@inngest/components/utils/appsParser';

import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query Sync($envID: ID!, $externalAppID: String!, $syncID: ID!) {
    environment: workspace(id: $envID) {
      app: appByExternalID(externalID: $externalAppID) {
        id
        externalID
        name
        method
      }
    }
    sync: deploy(id: $syncID) {
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
      removedFunctions: removedFunctions {
        id
        name
        slug
      }
      syncedFunctions: deployedFunctions {
        id
        name
        slug
      }
      url
      vercelDeploymentID
      vercelDeploymentURL
      vercelProjectID
      vercelProjectURL
    }
  }
`);

export function useSync({
  envID,
  externalAppID,
  syncID,
}: {
  envID: string;
  externalAppID: string;
  syncID: string;
}) {
  const res = useGraphQLQuery({
    query,
    variables: { envID, externalAppID, syncID },
  });

  if (res.data) {
    const sync = {
      ...res.data.sync,
      lastSyncedAt: new Date(res.data.sync.lastSyncedAt),
      framework: transformFramework(res.data.sync.framework),
      platform: transformPlatform(res.data.sync.platform),
      sdkLanguage: transformLanguage(res.data.sync.sdkLanguage),
    };

    return {
      ...res,
      data: {
        ...res.data,
        sync,
      },
    };
  }

  return res;
}
