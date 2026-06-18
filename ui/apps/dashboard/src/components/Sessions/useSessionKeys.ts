import { useDeferredValue } from 'react';
import { type SessionKey } from '@inngest/components/types/session';
import { useQuery } from '@tanstack/react-query';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';

const sessionKeysQuery = `
  query SessionKeys($workspaceID: ID!, $search: String) {
    environment: workspace(id: $workspaceID) {
      sessionKeys(search: $search) {
        sessionKey
        createdAt
      }
    }
  }
`;

type SessionKeysQueryResult = {
  environment: {
    sessionKeys: Array<{
      sessionKey: string;
      createdAt: string;
    }>;
  } | null;
};

type SessionKeysQueryVariables = {
  workspaceID: string;
  search: string | null;
};

export function useSessionKeys(search: string) {
  const client = useClient();
  const envID = useEnvironment().id;
  const deferredSearch = useDeferredValue(search.trim());

  return useQuery({
    queryKey: ['sessionKeys', envID, deferredSearch],
    queryFn: async (): Promise<SessionKey[]> => {
      const result = await client
        .query<
          SessionKeysQueryResult,
          SessionKeysQueryVariables
        >(sessionKeysQuery, { workspaceID: envID, search: deferredSearch || null }, { requestPolicy: 'network-only' })
        .toPromise();

      if (result.error) throw result.error;

      return (result.data?.environment?.sessionKeys ?? []).map((key) => ({
        sessionKey: key.sessionKey,
        createdAt: key.createdAt,
      }));
    },
    refetchOnWindowFocus: false,
  });
}
