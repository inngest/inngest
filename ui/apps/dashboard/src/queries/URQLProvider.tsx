'use client';

import { useMemo } from 'react';
import { useAuth } from '@clerk/nextjs';
import { authExchange } from '@urql/exchange-auth';
import { requestPolicyExchange } from '@urql/exchange-request-policy';
import { Provider, cacheExchange, createClient, fetchExchange } from 'urql';

export default function URQLProvider({ children }: { children: React.ReactNode }) {
  const { getToken } = useAuth();

  const urqlClient = useMemo(() => {
    return createClient({
      url: `${process.env.NEXT_PUBLIC_API_URL}/gql`,
      exchanges: [
        requestPolicyExchange({
          // The amount of time in ms that has to go by before we upgrade the operation's request policy to `cache-and-network`.
          ttl: 30_000, // 30 seconds (same value as Next.js’ Full Route Cache)
          // Only upgrade if the request policy is not `cache-only`
          shouldUpgrade: (operation) => operation.context.requestPolicy !== 'cache-only',
        }),
        cacheExchange,
        authExchange(async (utils) => {
          let sessionToken = await getToken();
          return {
            addAuthToOperation: (operation) => {
              if (!sessionToken) return operation;
              return utils.appendHeaders(operation, {
                Authorization: `Bearer ${sessionToken}`,
              });
            },
            didAuthError: (error) =>
              error.response.status === 401 ||
              error.graphQLErrors.some((e) => e.extensions.code === 'UNAUTHENTICATED'),
            refreshAuth: async () => {
              sessionToken = await getToken({ skipCache: true });
            },
          };
        }),
        fetchExchange,
      ],
    });
  }, [getToken]);

  return <Provider value={urqlClient}>{children}</Provider>;
}
