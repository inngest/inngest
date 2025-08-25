import { useMemo } from 'react';
import { ClerkProvider, useAuth } from '@clerk/tanstack-react-start';
import * as Sentry from '@sentry/nextjs';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Outlet, createRootRoute } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { authExchange } from '@urql/exchange-auth';
import { requestPolicyExchange } from '@urql/exchange-request-policy';
import { retryExchange } from '@urql/exchange-retry';
import {
  Provider as URQLProvider,
  cacheExchange,
  createClient,
  fetchExchange,
  mapExchange,
} from 'urql';

// Create QueryClient for TanStack Query (used by components library)
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      gcTime: 1000 * 60 * 10, // 10 minutes
    },
  },
});

// Inner component that can use auth hooks
function RootWithAuth() {
  const { getToken, signOut } = useAuth();

  const urqlClient = useMemo(() => {
    return createClient({
      url: `${process.env.NEXT_PUBLIC_API_URL}/gql`,
      fetchOptions: {
        credentials: 'include',
      },
      exchanges: [
        requestPolicyExchange({
          ttl: 30_000, // 30 seconds
        }),
        mapExchange({
          onError: (error, operation) => {
            const networkError = error.networkError;
            if (networkError) {
              // Log error to Sentry
              Sentry.captureException(networkError, {
                tags: {
                  component: 'URQL',
                },
                extra: {
                  operation: operation.query,
                  variables: operation.variables,
                },
              });
            }

            // Handle authentication errors
            const isUnauthenticated = error.graphQLErrors.some((e) => {
              return (
                e.extensions.code === 'UNAUTHENTICATED' || e.message.includes('Unauthenticated')
              );
            });

            if (isUnauthenticated) {
              signOut();
            }
          },
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
            didAuthError: (error) => {
              return error.graphQLErrors.some((e) => e.extensions.code === 'UNAUTHENTICATED');
            },
            refreshAuth: async () => {
              sessionToken = await getToken({ skipCache: true });
            },
          };
        }),
        retryExchange({
          retryIf: (error) => {
            const networkErrorCodes = [408, 429, 502, 503, 504];
            return (
              !!error.networkError &&
              networkErrorCodes.includes(parseInt(error.networkError.message))
            );
          },
        }),
        fetchExchange,
      ],
    });
  }, [getToken, signOut]);

  return (
    <QueryClientProvider client={queryClient}>
      <URQLProvider value={urqlClient}>
        <div className="min-h-screen bg-gray-50">
          <Outlet />
        </div>
        <TanStackRouterDevtools />
      </URQLProvider>
    </QueryClientProvider>
  );
}

// Root route component with ClerkProvider
const RootComponent = () => (
  <ClerkProvider publishableKey={process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY}>
    <RootWithAuth />
  </ClerkProvider>
);

export const Route = createRootRoute({
  component: RootComponent,
});
