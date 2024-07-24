import { useQuery, type UseQueryResponse } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { type GetDeployssQuery } from '@/gql/graphql';

const GetDeploysDocument = graphql(`
  query GetDeployss($environmentID: ID!) {
    deploys(workspaceID: $environmentID) {
      id
      appName
      authorID
      checksum
      createdAt
      error
      framework
      metadata
      sdkLanguage
      sdkVersion
      status

      deployedFunctions {
        id
        name
      }

      removedFunctions {
        id
        name
      }
    }
  }
`);

// TODO - Support pagination/cursors
export const useDeploys = (): UseQueryResponse<GetDeployssQuery, { environmentID: string }> => {
  const environment = useEnvironment();
  const [result, refetch] = useQuery({
    query: GetDeploysDocument,
    variables: {
      environmentID: environment.id,
    },
  });

  return [{ ...result, fetching: result.fetching }, refetch];
};
