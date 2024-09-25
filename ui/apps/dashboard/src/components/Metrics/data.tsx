import 'server-only';
import { cache } from 'react';

import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';

const MetricsLookupDocument = graphql(`
  query MetricsLookups($envSlug: String!, $page: Int, $pageSize: Int) {
    envBySlug(slug: $envSlug) {
      apps {
        externalID
        id
        name
        isArchived
      }
      workflows @paginated(perPage: $pageSize, page: $page) {
        data {
          name
          id
          slug
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

export const preloadMetricsLookups = (envSlug: string) => {
  void getMetricsLookups(envSlug);
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
    envBySlug: { apps: []; workflows: { data: []; page: { page: number; totalPages: number } } };
  }>(MetricsLookupDocument, { envSlug, page, pageSize });

export const getMetricsLookups = cache(async (envSlug: string) => {
  const page = 1;
  const pageSize = 1000;
  const results = await fetchLookups({ envSlug, page, pageSize });

  const totalPages = results.envBySlug.workflows.page.totalPages || 1;

  if (totalPages === 1) {
    return results;
  }

  for (let p = 1; p <= totalPages; p++) {
    const pageResult = await fetchLookups({ envSlug, page: p, pageSize });
    results.envBySlug.workflows.data = {
      ...results.envBySlug.workflows.data,
      ...pageResult.envBySlug.workflows.data,
    };
  }

  return results;
});
