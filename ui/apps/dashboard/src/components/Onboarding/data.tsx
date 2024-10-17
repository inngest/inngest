import 'server-only';
import { SyncNewAppDocument } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/sync-new/ManualSync';
import { type SyncResponse } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getProductionEnvironment } from '@/queries/server-only/getEnvironment';

export const syncNewApp = async (appURL: string) => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<{ syncNewApp: SyncResponse }>(SyncNewAppDocument, {
    envID: environment.id,
    appURL: appURL,
  });
};
