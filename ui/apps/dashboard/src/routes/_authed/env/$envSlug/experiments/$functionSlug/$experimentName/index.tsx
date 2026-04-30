import { createFileRoute } from '@tanstack/react-router';

import NotFound from '@/components/Error/NotFound';
import { ExperimentDetailPage } from '@/components/Experiments/ExperimentDetailPage';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useFunction } from '@/queries/functions';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/experiments/$functionSlug/$experimentName/',
)({
  component: ExperimentDetailRoute,
});

function ExperimentDetailRoute() {
  const { functionSlug, experimentName } = Route.useParams();
  const experimentsEnabled = useBooleanFlag('experimentation-steps');

  const [{ data, fetching }] = useFunction({
    functionSlug: decodeURIComponent(functionSlug),
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
