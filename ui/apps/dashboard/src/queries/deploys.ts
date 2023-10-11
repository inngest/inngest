import { useQuery, type UseQueryResponse } from 'urql';

import { graphql } from '@/gql';
import { type GetDeployssQuery } from '@/gql/graphql';
import { useEnvironment } from '@/queries/environments';

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

type UseDeploysParams = {
  environmentSlug: string;
};

// TODO - Support pagination/cursors
export const useDeploys = ({
  environmentSlug,
}: UseDeploysParams): UseQueryResponse<GetDeployssQuery, { environmentID: string }> => {
  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });
  const [result, refetch] = useQuery({
    query: GetDeploysDocument,
    variables: {
      environmentID: environment?.id!,
    },
    pause: !environment?.id,
  });

  return [{ ...result, fetching: isFetchingEnvironment || result.fetching }, refetch];
};
