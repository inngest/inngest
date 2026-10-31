import { useDeferredValue, useMemo } from 'react';
import type { SessionKey } from '@inngest/components/types/session';
import { useInfiniteQuery } from '@tanstack/react-query';

import { client } from '@/store/baseApi';

const sessionKeysQuery = `
  query SessionKeys($search: String, $after: String) {
    sessionKeys(search: $search, after: $after) {
      nodes {
        sessionKey
        createdAt
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
`;

type SessionKeysQueryResult = {
  sessionKeys: {
    nodes: Array<{
      sessionKey: string;
      createdAt: string;
    }>;
    pageInfo: { hasNextPage: boolean; endCursor: string | null };
  };
};

export function useSessionKeys(search: string) {
  const deferredSearch = useDeferredValue(search.trim());

  const query = useInfiniteQuery({
    queryKey: ['sessionKeys', deferredSearch],
    queryFn: async ({ pageParam }: { pageParam: string | null }) => {
      const result = await client.request<SessionKeysQueryResult>(
        sessionKeysQuery,
        {
          search: deferredSearch || null,
          after: pageParam,
        },
      );
      return result.sessionKeys;
    },
    initialPageParam: null,
    getNextPageParam: (lastPage) =>
      lastPage.pageInfo.hasNextPage
        ? lastPage.pageInfo.endCursor ?? undefined
        : undefined,
    refetchOnWindowFocus: false,
  });

  const data = useMemo(
    (): SessionKey[] | undefined =>
      query.data?.pages.flatMap((page) =>
        page.nodes.map((key) => ({
          sessionKey: key.sessionKey,
          createdAt: key.createdAt,
        })),
      ),
    [query.data],
  );

  return { ...query, data };
}
