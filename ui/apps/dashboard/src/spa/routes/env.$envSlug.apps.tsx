import { useEffect, useState } from 'react';
import { SkeletonCard } from '@inngest/components/Apps/AppCard';
import { Button } from '@inngest/components/Button';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link/Link';
import { RiAddLine } from '@remixicon/react';
import { createFileRoute, getRouteApi } from '@tanstack/react-router';

import { useApps } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/useApps';
import { useLatestUnattachedSync } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/useUnattachedSyncs';
import AppCards from '@/components/Apps/AppCards';
import {
  EmptyActiveCard,
  EmptyArchivedCard,
  EmptyOnboardingCard,
} from '@/components/Apps/EmptyAppsCard';
import { StatusMenu } from '@/components/Apps/StatusMenu';
import { UnattachedSyncsCard } from '@/components/Apps/UnattachedSyncsCard';
import { getProdApps } from '@/components/Onboarding/actions';
import TanStackAppFAQ from '@/spa/components/TanStackAppFAQ';
import { useEnvironmentContext } from '@/spa/contexts/EnvironmentContext';
import { staticSlugs } from '@/utils/environments';
import { pathCreator } from '@/utils/urls';

const routeApi = getRouteApi('/env/$envSlug/apps' as any);

const AppInfo = () => (
  <Info
    text="Apps map directly to your products or services."
    action={
      <Link href="https://www.inngest.com/docs/apps" target="_blank">
        Learn how apps work
      </Link>
    }
  />
);

type LoadingState = {
  hasProductionApps: boolean;
  isLoading: boolean;
};

function TanStackRouterApps({ isArchived = false }: { isArchived?: boolean }) {
  const { environment } = useEnvironmentContext();

  // Call hooks unconditionally at the top - use environment.id or empty string as fallback
  const unattachedSyncRes = useLatestUnattachedSync({
    envID: environment?.id || '',
  });
  const appsRes = useApps({
    envID: environment?.id || '',
    isArchived,
  });

  if (!environment) {
    return (
      <div className="mb-4 flex items-center justify-center">
        <div className="w-full">
          <SkeletonCard />
        </div>
      </div>
    );
  }

  if (unattachedSyncRes.error) {
    console.error(unattachedSyncRes.error);
  }

  if (appsRes.error) {
    throw appsRes.error;
  }

  if (appsRes.isPending) {
    return (
      <div className="mb-4 flex items-center justify-center">
        <div className="w-full">
          <SkeletonCard />
        </div>
      </div>
    );
  }

  const apps = appsRes.data;
  const hasApps = apps.length > 0;

  if (unattachedSyncRes.data && !isArchived) {
    return (
      <div className="flex items-center justify-center">
        <div className="w-full">
          <UnattachedSyncsCard envSlug={environment.slug} latestSyncTime={unattachedSyncRes.data} />
        </div>
      </div>
    );
  }

  if (!hasApps && !isArchived) {
    return (
      <div className="flex items-center justify-center">
        <div className="w-full">
          <EmptyActiveCard envSlug={environment.slug} />
          <TanStackAppFAQ />
        </div>
      </div>
    );
  }

  if (!hasApps && isArchived) {
    return (
      <div className="flex items-center justify-center">
        <div className="w-full">
          <EmptyArchivedCard />
        </div>
      </div>
    );
  }

  if (hasApps) {
    return (
      <div className="flex items-center justify-center">
        <div className="w-full">
          <AppCards apps={apps} envSlug={environment.slug} />
        </div>
      </div>
    );
  }

  return <div>No content to show</div>;
}

async function fetchInitialData(): Promise<LoadingState> {
  try {
    const result = await getProdApps();
    if (!result) {
      return { hasProductionApps: true, isLoading: false };
    }
    const { apps, unattachedSyncs } = result;
    const hasAppsOrUnattachedSyncs = apps.length > 0 || unattachedSyncs.length > 0;
    return { hasProductionApps: hasAppsOrUnattachedSyncs, isLoading: false };
  } catch (error) {
    console.error('Error fetching production apps', error);
    return { hasProductionApps: false, isLoading: false };
  }
}

function AppsPage() {
  const { envSlug } = (routeApi as any).useParams();
  const search = (routeApi as any).useSearch();
  const { environment, loading: envLoading } = useEnvironmentContext();

  const [{ hasProductionApps, isLoading }, setState] = useState<LoadingState>({
    hasProductionApps: true,
    isLoading: true,
  });

  const isArchived = search?.archived === 'true';

  useEffect(() => {
    fetchInitialData().then((data) => {
      setState(data);
    });
  }, []);

  if (envLoading || !environment) {
    return (
      <div className="mt-16 flex place-content-center">
        <div className="rounded-lg border border-blue-200 bg-blue-50 p-6 text-center">
          <h2 className="text-lg font-semibold text-blue-900">Loading apps...</h2>
        </div>
      </div>
    );
  }

  const displayOnboarding = envSlug === staticSlugs.production && !hasProductionApps;

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Apps' }]}
        infoIcon={<AppInfo />}
        action={
          (!isArchived || displayOnboarding) && (
            <Button
              kind="primary"
              label="Sync new app"
              href={pathCreator.createApp({ envSlug })}
              icon={<RiAddLine />}
              iconSide="left"
            />
          )
        }
      />
      <div className="bg-canvasBase h-full w-full">
        <div className="mx-auto flex h-full w-full max-w-4xl flex-col px-6 pb-4 pt-16">
          {isLoading ? (
            <div className="mb-4 flex items-center justify-center">
              <div className="mt-[50px] w-full max-w-4xl">
                <SkeletonCard />
              </div>
            </div>
          ) : (
            <>
              {displayOnboarding ? (
                <>
                  <EmptyOnboardingCard />
                  <TanStackAppFAQ />
                </>
              ) : (
                <>
                  <div className="relative flex w-full flex-row justify-start">
                    <StatusMenu archived={isArchived} envSlug={envSlug} />
                  </div>
                  <TanStackRouterApps isArchived={isArchived} />
                </>
              )}
            </>
          )}
        </div>
      </div>
    </>
  );
}

export const Route = createFileRoute('/env/$envSlug/apps')({
  component: AppsPage,
  validateSearch: (search: Record<string, unknown>) => {
    return {
      archived: search.archived as string,
    };
  },
});
