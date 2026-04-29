import { createFileRoute } from '@tanstack/react-router';

import NotFound from '@/components/Error/NotFound';
import { ExperimentDetailPage } from '@/components/Experiments/ExperimentDetailPage';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useFunction } from '@/queries/functions';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/functions/$slug/experiments/$experimentName/',
)({
  component: NestedExperimentComponent,
});

function NestedExperimentComponent() {
  const { slug, experimentName } = Route.useParams();
  const experimentsEnabled = useBooleanFlag('experimentation-steps');

  // Resolve the slug to a function ID. ExperimentDetailPage filters its
  // queries by function ID so two functions sharing an experiment name don't
  // collapse into a single set of metrics. The function detail header
  // (parent route) also issues this query, so urql will dedupe it.
  const [{ data, fetching }] = useFunction({
    functionSlug: decodeURIComponent(slug),
  });
  const functionID = data?.workspace.workflow?.id ?? null;

  if (experimentsEnabled.isReady && !experimentsEnabled.value) {
    return <NotFound />;
  }

  if (!fetching && functionID === null) {
    return <NotFound />;
  }

  if (functionID === null) {
    return null;
  }

  return (
    <ExperimentDetailPage
      experimentName={decodeURIComponent(experimentName)}
      functionID={functionID}
    />
  );
}
