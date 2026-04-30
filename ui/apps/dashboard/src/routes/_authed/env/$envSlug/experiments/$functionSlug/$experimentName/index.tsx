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

  const decodedFunctionSlug = decodeURIComponent(functionSlug);
  const [{ data, fetching }] = useFunction({
    functionSlug: decodedFunctionSlug,
  });
  const fn = data?.workspace.workflow;

  if (experimentsEnabled.isReady && !experimentsEnabled.value) {
    return <NotFound />;
  }

  if (!fetching && !fn) {
    return <NotFound />;
  }

  if (!fn) {
    return null;
  }

  return (
    <ExperimentDetailPage
      experimentName={decodeURIComponent(experimentName)}
      functionID={fn.id}
      functionName={fn.name}
      functionSlug={decodedFunctionSlug}
    />
  );
}
