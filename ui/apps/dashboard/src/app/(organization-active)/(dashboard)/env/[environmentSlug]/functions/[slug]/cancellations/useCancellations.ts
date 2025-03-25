import { useCallback, useEffect, useMemo, useState } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useClient } from 'urql';

import { graphql } from '@/gql';

const query = graphql(`
  query GetFnCancellations($after: String, $envSlug: String!, $fnSlug: String!) {
    env: envBySlug(slug: $envSlug) {
      fn: workflowBySlug(slug: $fnSlug) {
        cancellations(after: $after) {
          edges {
            cursor
            node {
              createdAt
              envID: environmentID
              id
              name
              queuedAtMax
              queuedAtMin
            }
          }
          pageInfo {
            hasNextPage
          }
        }
      }
    }
  }
`);

export function useCancellations({ envSlug, fnSlug }: { envSlug: string; fnSlug: string }) {
  const [hasNextPage, setHasNextPage] = useState(true);
  const [isInitiallyFetching, setIsInitiallyFetching] = useState(true);

  const client = useClient();

  const queryFn = useCallback(
    async ({ pageParam }: { pageParam: string | null }) => {
      const res = await client.query(query, {
        after: pageParam,
        envSlug,
        fnSlug,
      });
      if (res.error) {
        throw res.error;
      }
      if (!res.data) {
        throw new Error('No data');
      }

      if (!res.data.env) {
        throw new Error('environment not found');
      }
      if (!res.data.env.fn) {
        throw new Error('function not found');
      }

      setHasNextPage(res.data.env.fn.cancellations.pageInfo.hasNextPage);

      return res.data.env.fn.cancellations;
    },
    [client, envSlug, fnSlug]
  );

  const { data, fetchNextPage, isFetching } = useInfiniteQuery({
    queryKey: ['runs'],
    queryFn,
    refetchInterval: 2500,
    initialPageParam: null,
    getNextPageParam: (lastPage) => {
      const endCursor = lastPage.edges[lastPage.edges.length - 1]?.cursor;
      if (!endCursor) {
        return undefined;
      }

      return endCursor;
    },
  });

  const allData = useMemo(() => {
    if (!data?.pages) {
      return [];
    }
    if (data.pages.length === 0) {
      return [];
    }

    const out = [];
    for (const page of data.pages) {
      out.push(...page.edges.map((edge) => edge.node));
    }

    return out;
  }, [data?.pages]);

  useEffect(() => {
    if (data) {
      setIsInitiallyFetching(false);
    }
  }, [data]);

  return {
    data: allData,
    fetchNextPage,
    hasNextPage,
    isFetching,
    isInitiallyFetching,
  };
}
