import { useEffect, useState } from 'react';
import { Outlet, createFileRoute, getRouteApi } from '@tanstack/react-router';
import { useClient } from 'urql';

import { GetEnvironmentBySlugDocument } from '@/gql/graphql';
import TanStackLayout from '@/spa/components/TanStackLayout';
import { EnvironmentProvider } from '@/spa/contexts/EnvironmentContext';

// Create route API for type-safe hooks (with CLI limitations workaround)
const routeApi = getRouteApi('/env/$envSlug' as any);

function EnvironmentLayout() {
  const { envSlug } = (routeApi as any).useParams();
  const client = useClient(); // This will use the authenticated URQL client from component context
  const [env, setEnv] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadEnvironment() {
      try {
        setLoading(true);
        setError(null);

        const result = await client
          .query(GetEnvironmentBySlugDocument, { slug: envSlug }, { requestPolicy: 'network-only' })
          .toPromise();

        if (result.error) {
          throw new Error(result.error.message);
        }

        if (!result.data?.envBySlug) {
          throw new Error('Environment not found');
        }

        setEnv(result.data.envBySlug);
      } catch (err: any) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }

    loadEnvironment();
  }, [envSlug, client]);

  if (loading) {
    return (
      <div className="mt-16 flex place-content-center">
        <div className="rounded-lg border border-blue-200 bg-blue-50 p-6 text-center">
          <h2 className="text-lg font-semibold text-blue-900">Loading environment...</h2>
        </div>
      </div>
    );
  }

  if (error || !env) {
    return (
      <div className="mt-16 flex place-content-center">
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-center">
          <h2 className="text-lg font-semibold text-red-900">Environment not found</h2>
          {error && <p className="mt-2 text-sm text-gray-600">{error}</p>}
        </div>
      </div>
    );
  }

  return (
    <EnvironmentProvider environment={env} loading={loading} error={error}>
      <TanStackLayout>
        <Outlet />
      </TanStackLayout>
    </EnvironmentProvider>
  );
}

export const Route = createFileRoute('/env/$envSlug')({
  component: EnvironmentLayout,
});
