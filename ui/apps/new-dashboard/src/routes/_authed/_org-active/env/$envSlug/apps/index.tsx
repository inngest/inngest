import { getProdApps } from "@/components/Onboarding/actions";
import { staticSlugs } from "@/utils/environments";

import { EmptyOnboardingCard } from "@/components/Apps/EmptyAppsCard";
import { SkeletonCard } from "@inngest/components/Apps/AppCard";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";

import AppFAQ from "@/components/Apps/AppFAQ";
import { Apps } from "@/components/Apps/Apps";
import { StatusMenu } from "@/components/Apps/StatusMenu";

type AppsSearchParams = {
  archived?: string;
};

type LoadingState = {
  hasProductionApps: boolean;
  isLoading: boolean;
};

const fetchInitialData = async (): Promise<LoadingState> => {
  try {
    const result = await getProdApps();
    if (!result) {
      // In case of data fetching error, we don't wanna fail the page here
      return { hasProductionApps: true, isLoading: false };
    }
    const { apps, unattachedSyncs } = result;
    const hasAppsOrUnattachedSyncs =
      apps.length > 0 || unattachedSyncs.length > 0;
    return { hasProductionApps: hasAppsOrUnattachedSyncs, isLoading: false };
  } catch (error) {
    console.error("Error fetching production apps", error);
    return { hasProductionApps: false, isLoading: false };
  }
};

export const Route = createFileRoute("/_authed/_org-active/env/$envSlug/apps/")(
  {
    component: AppsPage,
    validateSearch: (search: Record<string, unknown>): AppsSearchParams => {
      return {
        archived: search?.archived as string | undefined,
      };
    },
  },
);

function AppsPage() {
  const { archived: archivedParam } = Route.useSearch();
  const { envSlug } = Route.useParams();

  const [{ hasProductionApps, isLoading }, setState] = useState<LoadingState>({
    hasProductionApps: true,
    isLoading: true,
  });

  const isArchived = archivedParam === "true";

  useEffect(() => {
    fetchInitialData().then((data) => {
      setState(data);
    });
  }, []);

  const displayOnboarding =
    envSlug === staticSlugs.production && !hasProductionApps;

  return (
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
  );
}
