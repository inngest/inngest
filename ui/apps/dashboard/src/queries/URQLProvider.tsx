'use client';

import { useMemo } from 'react';
import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { useAuth } from '@clerk/nextjs';
import * as Sentry from '@sentry/nextjs';
import { authExchange } from '@urql/exchange-auth';
import { requestPolicyExchange } from '@urql/exchange-request-policy';
import { retryExchange } from '@urql/exchange-retry';
import {
  CombinedError,
  Provider,
  cacheExchange,
  createClient,
  fetchExchange,
  mapExchange,
} from 'urql';

import SignInRedirectErrors from '@/app/(auth)/sign-in/[[...sign-in]]/SignInRedirectErrors';

/**
 * This is used to ensure that the URQL client is re-created (cache reset) whenever the user signs
 * out or switches organizations.
 * @param {React.ReactNode} children
 * @returns {JSX.Element}
 * @constructor
 */
export default function URQLProviderWrapper({ children }: { children: React.ReactNode }) {
  const { isSignedIn, orgId } = useAuth();

  return <URQLProvider key={`${isSignedIn}-${orgId}`}>{children}</URQLProvider>;
}

export function URQLProvider({ children }: { children: React.ReactNode }) {
  const { getToken, signOut } = useAuth();
  const router = useRouter();

  const client = useMemo(() => {
    return createClient({
      url: `${process.env.NEXT_PUBLIC_API_URL}/gql`,
      fetchOptions: {
        // Necessary to include HTTP-only cookies. This is used for non-Clerk
        // auth.
        credentials: 'include',
      },
      exchanges: [
        requestPolicyExchange({
          // The amount of time in ms that has to go by before we upgrade the operation's request policy to `cache-and-network`.
          ttl: 30_000, // 30 seconds (same value as Next.js’ Full Route Cache)
          // Only upgrade if the request policy is not `cache-only`
          shouldUpgrade: (operation) => operation.context.requestPolicy !== 'cache-only',
        }),
        cacheExchange,
        mapExchange({
          onError(error) {
            // Handle unauthenticated errors after (1) trying to refresh the token and (2) retrying the operation.
            if (isUnauthenticatedError(error)) {
              // Log to Sentry if it still fails after trying to refresh the token and retrying the operation.
              Sentry.captureException(error);
              signOut(() => {
                router.push(
                  `${process.env.NEXT_PUBLIC_SIGN_IN_PATH || '/sign-in'}?error=${
                    SignInRedirectErrors.Unauthenticated
                  }` as Route
                );
              });
            }
          },
        }),
        retryExchange({
          maxNumberAttempts: 3,
          retryIf: isUnauthenticatedError,
        }),
        authExchange(async (utils) => {
          let sessionToken = await getToken();
          return {
            addAuthToOperation: (operation) => {
              if (!sessionToken) return operation;
              return utils.appendHeaders(operation, {
                Authorization: `Bearer ${sessionToken}`,
              });
            },
            didAuthError: isUnauthenticatedError,
            refreshAuth: async () => {
              sessionToken = await getToken({ skipCache: true });
            },
          };
        }),
        fetchExchange,
      ],
    });
  }, [getToken, router, signOut]);

  return <Provider value={client}>{children}</Provider>;
}

function isUnauthenticatedError(error: CombinedError): boolean {
  return (
    error.response?.status === 401 ||
    error.graphQLErrors.some((e) => e.extensions.code === 'UNAUTHENTICATED')
  );
}
