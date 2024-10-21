import 'server-only';
import { graphql } from '@/gql';
import { type SyncResponse } from '@/gql/graphql';
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
