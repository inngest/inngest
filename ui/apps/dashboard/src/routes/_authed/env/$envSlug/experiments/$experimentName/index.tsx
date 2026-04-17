import { createFileRoute } from '@tanstack/react-router';

import NotFound from '@/components/Error/NotFound';
import { ExperimentDetailPage } from '@/components/Experiments/ExperimentDetailPage';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/experiments/$experimentName/',
)({
  component: ExperimentComponent,
});

function ExperimentComponent() {
  const { experimentName } = Route.useParams();
  const experimentsEnabled = useBooleanFlag('experimentation-steps');

  if (experimentsEnabled.isReady && !experimentsEnabled.value) {
    return <NotFound />;
  }

  return (
    <ExperimentDetailPage experimentName={decodeURIComponent(experimentName)} />
  );
}
