import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const query = graphql(`
  query AppNavData($envID: ID!, $externalAppID: String!) {
    environment: workspace(id: $envID) {
      app: appByExternalID(externalID: $externalAppID) {
        id
        isArchived
        isParentArchived
        latestSync {
          platform
          url
        }
        name
      }
    }
  }
`);

export function useNavData({ envID, externalAppID }: { envID: string; externalAppID: string }) {
  const res = useGraphQLQuery({
    query,
    variables: { envID, externalAppID },
  });

  if (res.data) {
    const { latestSync } = res.data.environment.app;
    if (latestSync?.url) {
      latestSync.url = removeSyncIDFromURL(latestSync.url);
    }

    return {
      ...res,
      data: {
        ...res.data.environment.app,
        latestSync,
      },
    };
  }

  return { ...res, data: undefined };
}

/**
 * Removes the sync ID from the URL. This is important since we want the sync to
 * be 100% new.
 */
function removeSyncIDFromURL(url: string): string {
  if (!url.startsWith('http')) {
    url = 'https://' + url;
  }

  const urlObj = new URL(url);
  urlObj.searchParams.delete('deployId');
  return urlObj.toString();
}
