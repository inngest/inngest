'use client';

import { useMemo } from 'react';
import { useAuth } from '@clerk/nextjs';
import { requestPolicyExchange } from '@urql/exchange-request-policy';
import { retryExchange } from '@urql/exchange-retry';
import {
  Provider,
  cacheExchange,
  createClient,
  fetchExchange,
  makeOperation,
  mapExchange,
  type Operation,
} from 'urql';

export default function URQLProvider({ children }: { children: React.ReactNode }) {
  const { getToken, isLoaded } = useAuth();

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
        retryExchange({
          maxNumberAttempts: 3,
          retryIf: (error) => {
            // Retry the operation if clerk is not loaded yet and we got an unauthorized error
            return (
              !isLoaded && error.graphQLErrors.some((e) => e.message?.includes('unauthorized'))
            );
          },
        }),
        mapExchange({
          // Append the Clerk session token to all operations (subscriptions, queries, mutations, teardowns)
          async onOperation(operation) {
            const sessionToken = await getToken();
            if (!sessionToken) return operation;
            return appendHeaders(operation, {
              Authorization: `Bearer ${sessionToken}`,
            });
          },
        }),
        fetchExchange,
      ],
      // TODO: Remove the following line once we have fully migrated to Clerk-based authentication
      fetchOptions: () => ({ credentials: 'include' }),
    });
  }, [getToken, isLoaded]);

  return <Provider value={urqlClient}>{children}</Provider>;
}

function appendHeaders(operation: Operation, headers: Record<string, string>): Operation {
  const fetchOptions =
    typeof operation.context.fetchOptions === 'function'
      ? operation.context.fetchOptions()
      : operation.context.fetchOptions || {};
  return makeOperation(operation.kind, operation, {
    ...operation.context,
    fetchOptions: {
      ...fetchOptions,
      headers: {
        ...fetchOptions.headers,
        ...headers,
      },
    },
  });
}
