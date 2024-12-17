import 'server-only';
import { graphql } from '@/gql';
import { type CdcConnectionInput, type CdcSetupResponse, type DeleteResponse } from '@/gql/graphql';
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

  return await graphqlAPI.request<{ cdcTestCredentials: CdcSetupResponse }>(testAuthDocument, {
    envID: environment.id,
    input: input,
  });
};

const testLogicalReplicationDocument = graphql(`
  mutation testReplication($input: CDCConnectionInput!, $envID: UUID!) {
    cdcTestLogicalReplication(input: $input, envID: $envID) {
      steps
      error
    }
  }
`);

export const testLogicalReplication = async (input: CdcConnectionInput) => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<{ cdcTestLogicalReplication: CdcSetupResponse }>(
    testLogicalReplicationDocument,
    {
      envID: environment.id,
      input: input,
    }
  );
};

const testAutoSetupDocument = graphql(`
  mutation testAutoSetup($input: CDCConnectionInput!, $envID: UUID!) {
    cdcAutoSetup(input: $input, envID: $envID) {
      steps
      error
    }
  }
`);

export const testAutoSetup = async (input: CdcConnectionInput) => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<{ cdcAutoSetup: CdcSetupResponse }>(testAutoSetupDocument, {
    envID: environment.id,
    input: input,
  });
};

const deleteConnDocument = graphql(`
  mutation cdcDelete($envID: UUID!, $id: UUID!) {
    cdcDelete(envID: $envID, id: $id) {
      ids
    }
  }
`);

export const deleteConn = async (id: string) => {
  const environment = await getProductionEnvironment();

  return await graphqlAPI.request<{ cdcDelete: DeleteResponse }>(deleteConnDocument as any, {
    envID: environment.id,
    id: id,
  });
};
