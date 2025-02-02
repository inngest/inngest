import { useCallback } from 'react';
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
        envs {
          name
          slug
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

  const queryFn = useCallback(async () => {
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
  }, [client, envSlug, term]);

  return useQuery({
    queryKey: ['quick-search', term, envSlug],
    queryFn,
    enabled: Boolean(term),
    gcTime: 0,
  });
}
