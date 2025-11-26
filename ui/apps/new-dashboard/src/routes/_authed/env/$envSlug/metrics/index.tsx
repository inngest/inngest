import { MetricsActionMenu } from "@/components/Metrics/ActionMenu";
import { Dashboard } from "@/components/Metrics/Dashboard";
import { Header } from "@inngest/components/Header/NewHeader";
import { RefreshButton } from "@inngest/components/Refresh/NewRefreshButton";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authed/env/$envSlug/metrics/")({
  component: MetricsComponent,
});

function MetricsComponent() {
  const { envSlug } = Route.useParams();
  return (
    <>
      <Header
        breadcrumb={[{ text: "Metrics" }]}
        action={
          <div className="flex flex-row items-center justify-end gap-x-1">
            <RefreshButton />
            <MetricsActionMenu />
          </div>
        }
      />
      <div id="chart-tooltip" className="z-[1000]" />
      <div className="bg-canvasBase mx-auto flex h-full w-full flex-col">
        <Dashboard envSlug={envSlug} />
      </div>
    </>
  );
}
