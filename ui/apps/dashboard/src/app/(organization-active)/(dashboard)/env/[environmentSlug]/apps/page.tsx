'use client';

import { useEffect, useState } from 'react';
import { NewButton } from '@inngest/components/Button';
import { Header } from '@inngest/components/Header/Header';
import { NewLink } from '@inngest/components/Link/Link';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiAddLine, RiQuestionLine } from '@remixicon/react';

import { StatusMenu } from '@/components/Apps/StatusMenu';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import EmptyAppsCard from '@/components/Onboarding/EmptyAppsCard';
import { getProdApps } from '@/components/Onboarding/actions';
import { staticSlugs } from '@/utils/environments';
import { pathCreator } from '@/utils/urls';
import { Apps } from './Apps';

const AppInfo = () => (
  <Tooltip>
    <TooltipTrigger>
      <RiQuestionLine className="text-subtle h-[18px] w-[18px]" />
    </TooltipTrigger>
    <TooltipContent
      side="right"
      sideOffset={2}
      className="border-muted text-muted text-md mt-6 flex flex-col rounded-lg border p-0 text-sm"
    >
      <div className="border-b px-4 py-2 ">Apps map directly to your products or services.</div>

      <div className="px-4 py-2">
        <NewLink href="https://www.inngest.com/docs/apps" size="small">
          Learn how apps work
        </NewLink>
      </div>
    </TooltipContent>
  </Tooltip>
);

export default function AppsPage({
  params: { environmentSlug: envSlug },
  searchParams: { archived },
}: {
  params: { environmentSlug: string };
  searchParams: { archived: string };
}) {
  const [hasProductionApps, setHasProductionApps] = useState(false);
  const isArchived = archived === 'true';
  const { value: onboardingFlow } = useBooleanFlag('onboarding-flow-cloud');

  useEffect(() => {
    async function fetchProductionApps() {
      try {
        const apps = await getProdApps();
        setHasProductionApps(apps.length > 0);
      } catch (error) {
        console.error('Error fetching production apps', error);
        setHasProductionApps(false);
      }
    }

    fetchProductionApps();
  }, []);

  const displayOnboarding =
    envSlug === staticSlugs.production && onboardingFlow && !hasProductionApps;

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Apps' }]}
        infoIcon={<AppInfo />}
        action={
          (!isArchived || displayOnboarding) && (
            <NewButton
              kind="primary"
              label="Sync new app"
              href={pathCreator.createApp({ envSlug })}
              icon={<RiAddLine />}
              iconSide="left"
            />
          )
        }
      />
      <div className="bg-canvasBase mx-auto flex h-full w-full max-w-[1200px] flex-col px-6 pt-16">
        {displayOnboarding ? (
          <EmptyAppsCard />
        ) : (
          <>
            <div className="relative flex w-full flex-row justify-start">
              <StatusMenu archived={isArchived} envSlug={envSlug} />
            </div>
            <Apps isArchived={isArchived} />
          </>
        )}
      </div>
    </>
  );
}
