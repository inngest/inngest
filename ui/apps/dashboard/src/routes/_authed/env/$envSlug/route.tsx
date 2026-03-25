import { ArchivedEnvBanner } from '@/components/Environments/ArchivedEnvBanner';
import { EnvironmentProvider } from '@/components/Environments/environment-context';
import { SharedContextProvider } from '@/components/SharedContext/SharedContextProvider';
import {
  hasDeepLinkParams,
  resolveDashboardDeepLink,
  stripDeepLinkParams,
} from '@/lib/deepLinks';
import { validateDashboardDeepLinkSearch } from '@/lib/deepLinkSearch';
import { jwtAuth } from '@/lib/auth';
import { Alert } from '@inngest/components/Alert';
import {
  createFileRoute,
  notFound,
  Outlet,
  redirect,
  useLocation,
} from '@tanstack/react-router';
import { useEffect } from 'react';

import { getEnvironment } from '@/queries/server/getEnvironment';

const NotFound = () => (
  <div className="mt-16 flex place-content-center">
    <Alert severity="warning">Environment not found.</Alert>
  </div>
);

export const Route = createFileRoute('/_authed/env/$envSlug')({
  component: EnvLayout,
  notFoundComponent: NotFound,
  validateSearch: validateDashboardDeepLinkSearch,
  beforeLoad: async ({ location, search }) => {
    if (!hasDeepLinkParams(search)) {
      return;
    }

    const result = await resolveDashboardDeepLink({
      data: {
        ...search,
        isJWTAuth: await jwtAuth(),
      },
    });

    if (result.status === 'invalid') {
      throw notFound({ data: { error: 'Deep link invalid or expired' } });
    }

    if (result.status === 'valid' && result.shouldSwitchOrganization) {
      throw redirect({
        to: '/switch-organization',
        search: {
          organization_id: result.organizationId,
          redirect_url: stripDeepLinkParams(location.href),
        },
      });
    }
  },
  loader: async ({ params }) => {
    const env = await getEnvironment({
      data: { environmentSlug: params.envSlug },
    });

    if (params.envSlug && !env) {
      throw notFound({ data: { error: 'Environment not found' } });
    }

    return {
      env,
    };
  },
});

function EnvLayout() {
  const { env } = Route.useLoaderData();
  const search = Route.useSearch();
  const location = useLocation();

  return (
    <>
      <DeepLinkCleanup
        shouldCleanup={hasDeepLinkParams(search)}
        href={stripDeepLinkParams(location.href)}
      />
      <ArchivedEnvBanner env={env} />
      <EnvironmentProvider env={env}>
        <SharedContextProvider>
          <Outlet />
        </SharedContextProvider>
      </EnvironmentProvider>
    </>
  );
}

function DeepLinkCleanup({
  href,
  shouldCleanup,
}: {
  href: string;
  shouldCleanup: boolean;
}) {
  useEffect(() => {
    if (!shouldCleanup) {
      return;
    }

    window.history.replaceState(window.history.state, '', href);
  }, [href, shouldCleanup]);

  return null;
}
