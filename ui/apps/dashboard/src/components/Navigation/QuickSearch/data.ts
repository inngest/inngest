import { useQuery } from '@tanstack/react-query';
import { useClient } from 'urql';

import { graphql } from '@/gql';

const quickSearchQuery = graphql(`
  query QuickSearch($term: String!, $envSlug: String!) {
    account {
      quickSearch(term: $term, envSlug: $envSlug) {
        apps {
          name
        }
        event {
          id
          name
        }
        eventTypes {
          name
        }
        functions {
          name
          slug
        }
        run {
          id
        }
      }
    }
  }
`);

export function useQuickSearch({ envSlug, term }: { envSlug: string; term: string }) {
  const client = useClient();

  return useQuery({
    queryKey: ['quick-search', term, envSlug],
    queryFn: async () => {
      const res = await client.query(quickSearchQuery, {
        envSlug,
        term,
      });
      if (res.error) {
        throw res.error;
      }
      if (!res.data) {
        throw new Error('No data');
      }
      return res.data.account.quickSearch;
    },
    enabled: Boolean(term),
    // gcTime: 0,
  });
}
