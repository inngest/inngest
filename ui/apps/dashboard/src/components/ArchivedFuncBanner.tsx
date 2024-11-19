'use client';

import { Banner } from '@inngest/components/Banner';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const Query = graphql(`
  query GetArchivedFuncBannerData($envID: ID!, $funcID: ID!) {
    environment: workspace(id: $envID) {
      function: workflow(id: $funcID) {
        id
        archivedAt
      }
    }
  }
`);

type Props = {
  funcID: string;
};

export function ArchivedFuncBanner({ funcID }: Props) {
  const env = useEnvironment();
  const { data, error } = useGraphQLQuery({
    query: Query,
    variables: {
      envID: env.id,
      funcID,
    },
  });
  if (error) {
    console.error(error);
    return null;
  }

  if (!data?.environment.function?.archivedAt) {
    return null;
  }

  if (env.isArchived) {
    // We don't want both this banner and the env banner
    return null;
  }

  return (
    <Banner severity="warning">
      <span className="font-semibold">Function is archived.</span> Unarchive it by enabling it in
      your app and resyncing.
    </Banner>
  );
}
