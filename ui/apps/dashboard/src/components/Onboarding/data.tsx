import 'server-only';
import { cache } from 'react';

import { graphql } from '@/gql';
import {
  type App,
  type Deploy,
  type GetVercelAppsQuery,
  type InvokeFunctionMutation,
  type InvokeFunctionMutationVariables,
  type ProductionAppsQuery,
  type SyncResponse,
} from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getProductionEnvironment } from '@/queries/server-only/getEnvironment';

export const SyncOnboardingAppDocument = graphql(`
  mutation SyncOnboardingApp($appURL: String!, $envID: UUID!) {
    syncNewApp(appURL: $appURL, envID: $envID) {
      app {
        externalID
        id
      }
      error {
        code
        data
        message
      }
    }
  }
`);

export const syncNewApp = async (appURL: string) => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<{ syncNewApp: SyncResponse }>(SyncOnboardingAppDocument, {
    envID: environment.id,
    appURL: appURL,
  });
};

export const InvokeFunctionOnboardingDocument = graphql(`
  mutation InvokeFunctionOnboarding($envID: UUID!, $data: Map, $functionSlug: String!, $user: Map) {
    invokeFunction(envID: $envID, data: $data, functionSlug: $functionSlug, user: $user)
  }
`);

export const invokeFn = async ({
  functionSlug,
  user,
  data,
}: Pick<InvokeFunctionMutationVariables, 'data' | 'functionSlug' | 'user'>) => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<{ invokeFunction: InvokeFunctionMutation }>(
    InvokeFunctionOnboardingDocument,
    {
      envID: environment.id,
      functionSlug: functionSlug,
      user: user,
      data: data,
    }
  );
};

export const InvokeFunctionLookupDocument = graphql(`
  query InvokeFunctionLookup($envSlug: String!, $page: Int, $pageSize: Int) {
    envBySlug(slug: $envSlug) {
      workflows @paginated(perPage: $pageSize, page: $page) {
        data {
          name
          id
          slug
          current {
            triggers {
              eventName
            }
          }
        }
        page {
          page
          totalPages
          perPage
        }
      }
    }
  }
`);

export const preloadInvokeFunctionLookups = (envSlug: string) => {
  void getInvokeFunctionLookups(envSlug);
};

const fetchLookups = async ({
  envSlug,
  page,
  pageSize,
}: {
  envSlug: string;
  page: number;
  pageSize: number;
}) =>
  graphqlAPI.request<{
    envBySlug: { workflows: { data: []; page: { page: number; totalPages: number } } };
  }>(InvokeFunctionLookupDocument, { envSlug, page, pageSize });

export const getInvokeFunctionLookups = cache(async (envSlug: string) => {
  const page = 1;
  const pageSize = 1000;
  const results = await fetchLookups({ envSlug, page, pageSize });

  const totalPages = results.envBySlug.workflows.page.totalPages || 1;

  if (totalPages === 1) {
    return results;
  }

  for (let p = 1; p <= totalPages; p++) {
    const pageResult = await fetchLookups({ envSlug, page: p, pageSize });
    results.envBySlug.workflows.data = [
      ...results.envBySlug.workflows.data,
      ...pageResult.envBySlug.workflows.data,
    ];
  }

  return results;
});

export const GetVercelAppsOnboardingDocument = graphql(`
  query GetVercelApps($envID: ID!) {
    environment: workspace(id: $envID) {
      unattachedSyncs(first: 1) {
        lastSyncedAt
        error
        url
        vercelDeploymentURL
      }
      apps {
        id
        name
        externalID
        isArchived
        latestSync {
          error
          id
          platform
          vercelDeploymentID
          vercelProjectID
          status
        }
      }
    }
  }
`);

export const getProductionApps = async () => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<ProductionAppsQuery>(GetProductionAppsDocument, {
    envID: environment.id,
  });
};

export const GetProductionAppsDocument = graphql(`
  query ProductionApps($envID: ID!) {
    environment: workspace(id: $envID) {
      apps {
        id
      }
      unattachedSyncs(first: 1) {
        lastSyncedAt
      }
    }
  }
`);

export const getVercelApps = async () => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<GetVercelAppsQuery>(GetVercelAppsOnboardingDocument, {
    envID: environment.id,
  });
};

export type VercelApp = Pick<App, 'id' | 'name' | 'externalID'> & {
  latestSync: Pick<
    Deploy,
    'id' | 'error' | 'platform' | 'vercelDeploymentID' | 'vercelProjectID' | 'status'
  > | null;
};

export type UnattachedSync = Pick<Deploy, 'lastSyncedAt' | 'error' | 'url' | 'vercelDeploymentURL'>;
