import { Skeleton } from "@inngest/components/Skeleton";
import { createFileRoute } from "@tanstack/react-router";
import { Suspense } from "react";

export const Route = createFileRoute("/_authed/_org-active/env/$env/apps/")({
  component: Apps,
});

function Apps() {
  //const { envID } = Route.useLoaderData();

  return (
    <div className="p-6">
      <Suspense fallback={<Skeleton className="w-full h-24" />}>
        <AppsData envID={"test"} />
      </Suspense>
    </div>
  );
}

function AppsData({ envID }: { envID: string }) {
  const { token } = Route.useRouteContext({
    select: ({ token }) => ({ token }),
  });

  // console.log("fetching apps using suspense query");
  // const { data } = useSuspenseQuery(appsQueryOptions(token, envID));

  return (
    <div className="flex flex-col gap-2 max-w-[800px] mx-auto">
      apps data coming soon...
      {/* {data.environment.apps.map((app) => (
        <div className="mb-6 " key={app.id}>
          <AppCard kind={"primary"}>
            <AppCard.Content
              url="/"
              app={app}
              pill={null}
              actions={
                <div className="items-top flex gap-2">
                  <Button appearance="outlined" label="View details" />
                </div>
              }
              workerCounter={0}
            />
          </AppCard>
        </div>
      ))} */}
    </div>
  );
}
