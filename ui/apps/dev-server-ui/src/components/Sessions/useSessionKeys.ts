import { useDeferredValue } from 'react';
import type { SessionKey } from '@inngest/components/types/session';
import { useQuery } from '@tanstack/react-query';

import { client } from '@/store/baseApi';

const sessionKeysQuery = `
  query SessionKeys($search: String) {
    sessionKeys(search: $search) {
      sessionKey
      createdAt
    }
  }
`;

type SessionKeysQueryResult = {
  sessionKeys: Array<{
    sessionKey: string;
    createdAt: string;
  }>;
};

export function useSessionKeys(search: string) {
  const deferredSearch = useDeferredValue(search.trim());

  return useQuery({
    queryKey: ['sessionKeys', deferredSearch],
    queryFn: async (): Promise<SessionKey[]> => {
      const result = await client.request<SessionKeysQueryResult>(
        sessionKeysQuery,
        {
          search: deferredSearch || null,
        },
      );

      return result.sessionKeys.map((key) => ({
        sessionKey: key.sessionKey,
        createdAt: key.createdAt,
      }));
    },
    refetchOnWindowFocus: false,
  });
}
