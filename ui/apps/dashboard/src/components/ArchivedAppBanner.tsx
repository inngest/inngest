'use client';

import { Banner } from '@inngest/components/Banner';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const Query = graphql(`
  query GetArchivedAppBannerData($envID: ID!, $externalAppID: String!) {
    environment: workspace(id: $envID) {
      app: appByExternalID(externalID: $externalAppID) {
        isArchived
      }
    }
  }
`);

type Props = {
  externalAppID: string;
};

export function ArchivedAppBanner({ externalAppID }: Props) {
  const env = useEnvironment();
  const { data, error } = useGraphQLQuery({
    query: Query,
    variables: {
      envID: env.id,
      externalAppID,
    },
  });
  if (error) {
    console.error(error);
    return null;
  }
  if (!data?.environment.app.isArchived) {
    return null;
  }

  if (env.isArchived) {
    // We don't want both this banner and the env banner
    return null;
  }

  return (
    <Banner severity="warning">
      <span className="font-semibold">App is archived.</span> You may unarchive it{' '}
      <Banner.Link
        severity="warning"
        className="inline-flex"
        href={pathCreator.app({ externalAppID, envSlug: env.slug })}
      >
        here
      </Banner.Link>
      .
    </Banner>
  );
}
