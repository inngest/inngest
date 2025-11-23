import { Alert } from "@inngest/components/Alert/NewAlert";
import { FunctionList } from "@inngest/components/Apps/NewFunctionList";
import { Button } from "@inngest/components/Button/NewButton";
import { Skeleton } from "@inngest/components/Skeleton/Skeleton";
import WorkerCounter from "@inngest/components/Workers/ConnectedWorkersDescription";
import { methodTypes } from "@inngest/components/types/app";
import { RiListCheck } from "@remixicon/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { AppGitCard } from "@/components/Apps/AppGitCard/AppGitCard";
import { AppInfoCard } from "@/components/Apps/AppInfoCard/AppInfoCard";
import { useApp } from "@/components/Apps/useApp";
import { useEnvironment } from "@/components/Environments/environment-context";
import WorkersSection from "@/components/Workers/WorkersSection";
import { useWorkersCount } from "@/components/Workers/useWorker";
import { pathCreator } from "@/utils/urls";

export const Route = createFileRoute("/_authed/env/$envSlug/apps/$externalID/")(
  {
    component: AppPage,
  },
);

function AppPage() {
  const { externalID, envSlug } = Route.useParams();
  const navigate = useNavigate();
  const getWorkerCount = useWorkersCount();

  const env = useEnvironment();

  const appRes = useApp({
    envID: env.id,
    externalAppID: externalID,
  });

  if (appRes.error) {
    if (!appRes.data) {
      throw appRes.error;
    }
    console.error(appRes.error);
  }

  if (appRes.isLoading && !appRes.data) {
    return (
      <div className="h-full overflow-y-auto">
        <div className="mx-auto my-12 flex w-full max-w-[1200px] flex-col gap-9 px-6">
          <div>
            <Skeleton className="mb-1 h-8 w-72" />
          </div>
          <AppInfoCard className="mb-4" loading />
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="mx-auto my-12 flex w-full max-w-[1200px] flex-col gap-9 px-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="mb-1 text-2xl">{appRes.data.name}</h2>
          </div>
          <Button
            appearance="outlined"
            iconSide="left"
            icon={<RiListCheck />}
            onClick={() =>
              navigate({
                to: pathCreator.appSyncs({
                  envSlug,
                  externalAppID: encodeURIComponent(externalID),
                }),
              })
            }
            label="See all syncs"
          />
        </div>

        {appRes.data.latestSync?.error && (
          <Alert className="mb-4" severity="error">
            {appRes.data.latestSync.error}
          </Alert>
        )}

        <AppInfoCard
          app={appRes.data}
          sync={appRes.data.latestSync}
          workerCounter={
            <WorkerCounter
              appID={appRes.data.id}
              getWorkerCount={getWorkerCount}
            />
          }
        />

        {appRes.data.latestSync && (
          <AppGitCard className="mb-4" sync={appRes.data.latestSync} />
        )}

        {appRes.data.method === methodTypes.Connect && (
          <WorkersSection appID={appRes.data.id} />
        )}

        <div>
          <h4 className="text-subtle mb-4 text-xl">
            Functions ({appRes.data.functions.length})
          </h4>
          <FunctionList
            envSlug={envSlug}
            functions={appRes.data.functions}
            pathCreator={{
              function: pathCreator.function,
              eventType: pathCreator.eventType,
            }}
          />
        </div>
      </div>
    </div>
  );
}
