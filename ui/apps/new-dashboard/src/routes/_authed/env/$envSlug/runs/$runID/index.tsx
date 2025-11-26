import { createFileRoute } from "@tanstack/react-router";

import { DashboardRunDetails } from "@/components/RunDetails/RunDetails";

export const Route = createFileRoute("/_authed/env/$envSlug/runs/$runID/")({
  component: RunDetailsPage,
});

function RunDetailsPage() {
  const { runID } = Route.useParams();

  return <DashboardRunDetails runID={runID} standalone={true} />;
}
