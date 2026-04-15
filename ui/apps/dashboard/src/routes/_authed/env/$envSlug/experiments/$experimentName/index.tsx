import { createFileRoute } from '@tanstack/react-router';

import { ExperimentDetailPage } from '@/components/Experiments/ExperimentDetailPage';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/experiments/$experimentName/',
)({
  component: ExperimentComponent,
});

function ExperimentComponent() {
  const { experimentName } = Route.useParams();
  return (
    <ExperimentDetailPage experimentName={decodeURIComponent(experimentName)} />
  );
}
