import { staticSlugs } from '@/utils/environments';

import { EmptyOnboardingCard } from '@/components/Apps/EmptyAppsCard';
import { SkeletonCard } from '@inngest/components/Apps/AppCard';
import { createFileRoute } from '@tanstack/react-router';

import AppFAQ from '@/components/Apps/AppFAQ';
import { Apps } from '@/components/Apps/Apps';
import { StatusMenu } from '@/components/Apps/StatusMenu';

import { getProdApps } from '@/queries/server/apps';
import { pathCreator } from '@/utils/urls';
import { Button } from '@inngest/components/Button/NewButton';
import { Header } from '@inngest/components/Header/NewHeader';
import { RiAddLine } from '@remixicon/react';
import { AppInfo } from './route';

export type AppsSearchParams = {
  archived?: string;
};

export const Route = createFileRoute('/_authed/env/$envSlug/apps/')({
  component: AppsPage,
  validateSearch: (search: Record<string, unknown>): AppsSearchParams => {
    return {
      archived: search?.archived as string | undefined,
    };
  },
  loader: async () => {
    try {
      const response = await getProdApps();
      if (!response) {
        // In case of data fetching error, we don't wanna fail the page here
        return { hasProductionApps: true, isLoading: false };
      }
      const { apps, unattachedSyncs } = response.environment;
      const hasAppsOrUnattachedSyncs =
        apps.length > 0 || unattachedSyncs.length > 0;
      return {
        hasProductionApps: hasAppsOrUnattachedSyncs,
        isLoading: false,
      };
    } catch (error) {
      console.error('Error fetching production apps', error);
      return { hasProductionApps: false, isLoading: false };
    }
  },
});

function AppsPage() {
  const { hasProductionApps = false, isLoading = true } = Route.useLoaderData();
  const { archived: archivedParam } = Route.useSearch();
  const { envSlug } = Route.useParams();

  const isArchived = archivedParam === 'true';

  const displayOnboarding =
    envSlug === staticSlugs.production && !hasProductionApps;

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Apps' }]}
        backNav
        infoIcon={<AppInfo />}
        action={
          (!isArchived || displayOnboarding) && (
            <Button
              kind="primary"
              label="Sync new app"
              to={pathCreator.createApp({ envSlug })}
              icon={<RiAddLine />}
              iconSide="left"
            />
          )
        }
      />
      <button
        onClick={() => {
          throw new Error('Test error');
        }}
      >
        Test Error
      </button>
      <div className="bg-canvasBase mx-auto flex h-full w-full max-w-4xl flex-col px-6 pb-4 pt-16">
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
                <AppFAQ />
              </>
            ) : (
              <>
                <div className="relative flex w-full flex-row justify-start">
                  <StatusMenu archived={isArchived} envSlug={envSlug} />
                </div>
                <Apps isArchived={isArchived} />
              </>
            )}
          </>
        )}
      </div>
    </>
  );
}
