import 'server-only';
import { graphql } from '@/gql';
import { type CdcConnectionInput, type CdcSetupResponse } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getProductionEnvironment } from '@/queries/server-only/getEnvironment';

const testAuthDocument = graphql(`
  mutation testCredentials($input: CDCConnectionInput!, $envID: UUID!) {
    cdcTestCredentials(input: $input, envID: $envID) {
      steps
      error
    }
  }
`);

export const testAuth = async (input: CdcConnectionInput) => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<CdcSetupResponse>(testAuthDocument, {
    envID: environment.id,
    input,
  });
};
